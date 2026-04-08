package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const inventoryKeyFmt = "inventory:%s" // inventory:{event_id}

// decrementIfAvailableScript atomically checks and decrements
// inventory in a single Redis operation.
//
// Returns 1 if inventory was available and decremented.
// Returns 0 if inventory was already at zero.
var decrementIfAvailableScript = redis.NewScript(`
	local count = redis.call("GET", KEYS[1])

	if not count then
		return -1  -- cache miss, fall back to Postgres
	end

	if tonumber(count) <= 0 then
		return -2  -- sold out
	end

	redis.call("DECRBY", KEYS[1], 1)
	return tonumber(count) - 1 -- return new count
`)

// InventoryStore manages the cacehd inventory count for events
type InventoryStore struct {
	client *Client
}

func NewInventoryStore(client *Client) *InventoryStore {
	return &InventoryStore{client: client}
}

func (s *InventoryStore) inventoryKey(eventID string) string {
	return fmt.Sprintf(inventoryKeyFmt, eventID)
}

// Get returns the current cached inventory count for an event.
// Returns -1 if the key does not exist (cache miss).
func (s *InventoryStore) Get(ctx context.Context, eventID string) (int64, error) {
	val, err := s.client.rdb.Get(ctx, s.inventoryKey(eventID)).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return -1, nil // cache miss,  not an error
		}
		return 0, fmt.Errorf("getting inventory count: %w", err)
	}
	return val, nil
}

// Set initializes or overwrites the inventory count using SET NX.
// Returns true if the key was set, false if it already existed.
// Used by the restart recovery worker to warm the cache without
// overwriting a valid existing count.
func (s *InventoryStore) Set(ctx context.Context, eventID string, count int64) (bool, error) {
	err := s.client.rdb.SetArgs(ctx, s.inventoryKey(eventID), count, redis.SetArgs{
		Mode: "NX", // only set if not exists
		TTL:  0,
	}).Err()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("setting inventory count: %w", err)
	}

	return true, nil
}

// ForceSet overwrites the inventory count unconditionally.
// Used by the reconciliation worker when healing divergence —
// at that point we know the Postgres value is correct and
// Redis must be updated regardless of current value.
func (s *InventoryStore) ForceSet(ctx context.Context, eventID string, count int64) error {
	if err := s.client.rdb.Set(ctx, s.inventoryKey(eventID), count, 0).Err(); err != nil {
		return fmt.Errorf("force setting inventory count: %w", err)
	}
	return nil
}

// Decrement atomically decrements the inventory count by 1.
// Called after a successful Postgres claim INSERT.
func (s *InventoryStore) Decrement(ctx context.Context, eventID string) (int64, error) {
	val, err := s.client.rdb.DecrBy(ctx, s.inventoryKey(eventID), 1).Result()
	if err != nil {
		return 0, fmt.Errorf("decrementing inventory count: %w", err)
	}
	return val, nil
}

// Increment atomically increments the inventory count by 1.
// Called after a claim is released or expired.
func (s *InventoryStore) Increment(ctx context.Context, eventID string) (int64, error) {
	val, err := s.client.rdb.IncrBy(ctx, s.inventoryKey(eventID), 1).Result()
	if err != nil {
		return 0, fmt.Errorf("incrementing inventory count: %w", err)
	}
	return val, nil
}

// DecrementIfAvailable atomically checks and decrements inventory.
//
// Returns:
//
//	 1 → decremented successfully, proceed with claim
//	-2 → sold out, reject claim
//	-1 → cache miss, fall back to Postgres
func (s *InventoryStore) DecrementIfAvailable(ctx context.Context, eventID string) (int64, error) {
	res, err := decrementIfAvailableScript.Run(
		ctx,
		s.client.rdb,
		[]string{s.inventoryKey(eventID)},
	).Int64()
	if err != nil {
		return 0, fmt.Errorf("decrementing inventory: %w", err)
	}
	return res, nil
}
