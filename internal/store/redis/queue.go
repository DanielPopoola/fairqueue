package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	waitingKeyFmt  = "waiting:%s"  // waiting:{event_id}
	admittedKeyFmt = "admitted:%s" // admitted:{event_id}
)

// joinScript atomically checks both ZSETs and adds the customer
// to the waiting queue only if they are not already present.
//
// Returns:
//
//	0 → already in waiting ZSET
//	1 → already in admitted ZSET
//	2 → successfully added
var joinScript = redis.NewScript(`
	local waiting  = KEYS[1]
	local admitted = KEYS[2]
	local customerID   = ARGV[1]
	local score    = tonumber(ARGV[2])

	if redis.call("ZSCORE", waiting, customerID) ~= false then
		return 0
	end

	if redis.call("ZSCORE", admitted, customerID) ~= false then
		return 1
	end

	redis.call("ZADD", waiting, score, customerID)
	return 2
`)

// admitBatchScript atomically moves the next n customers from
// waiting to admitted in a single Redis operation.
//
// Returns a flat list of admitted customerIDs,
// or an empty list if the waiting queue is empty.
var admitBatchScript = redis.NewScript(`
	local waiting   = KEYS[1]
	local admitted  = KEYS[2]
	local batchSize = tonumber(ARGV[1])
	local admScore  = tonumber(ARGV[2])

	local results = redis.call("ZPOPMIN", waiting, batchSize)

	if #results == 0 then
		return {}
	end

	local customerIDs = {}
	local zaddArgs = {}

	-- ZPOPMIN returns alternating member/score pairs:
	-- results[1]=customerID, results[2]=score, results[3]=customerID ...
	-- We only need the customerIDs (odd indices).
	for i = 1, #results, 2 do
		local customerID = results[i]
		table.insert(customerIDs, customerID)
		table.insert(zaddArgs, admScore)
		table.insert(zaddArgs, customerID)
	end

	-- Unpack zaddArgs as variadic args to ZADD.
	-- redis.call requires individual args, not a table,
	-- so we use unpack() to expand the table.
	redis.call("ZADD", admitted, unpack(zaddArgs))

	return customerIDs
`)

// evictExpiredScript atomically finds and removes customers whose
// admission window has expired, returning their customerIDs.
//
// KEYS[1] → admitted:{event_id}
// ARGV[1] → cutoff timestamp (nanoseconds) — anyone admitted
//
//	before this time has exceeded their window
var evictExpiredScript = redis.NewScript(`
	local admitted = KEYS[1]
	local cutoff   = ARGV[1]

	-- Find all members with score below the cutoff
	local expired = redis.call("ZRANGEBYSCORE", admitted, "0", cutoff)

	if #expired == 0 then
		return {}
	end

	-- Remove them all in one operation
	redis.call("ZREM", admitted, unpack(expired))

	return expired
`)

// QueueStore manages the virtual waiting queue using two Redis ZSETs.
type QueueStore struct {
	client *Client
}

func NewQueueStore(client *Client) *QueueStore {
	return &QueueStore{client: client}
}

func (s *QueueStore) waitingKey(eventID string) string {
	return fmt.Sprintf(waitingKeyFmt, eventID)
}

func (s *QueueStore) admittedKey(eventID string) string {
	return fmt.Sprintf(admittedKeyFmt, eventID)
}

// Join atomically checks both ZSETs and adds the customer to the
// waiting queue if they are not already present in either.
func (s *QueueStore) Join(ctx context.Context, eventID, customerID string) error {
	keys := []string{
		s.waitingKey(eventID),
		s.admittedKey(eventID),
	}
	args := []any{
		customerID,
		time.Now().UnixNano(),
	}

	res, err := joinScript.Run(ctx, s.client.rdb, keys, args...).Int()
	if err != nil {
		return fmt.Errorf("joining queue: %w", err)
	}

	switch res {
	case 0, 1:
		return domain.ErrAlreadyInQueue
	case 2:
		return nil
	default:
		return fmt.Errorf("unexpected join result: %d", res)
	}
}

// AdmitNextBatch atomically moves the next n longest-waiting customers
// from the waiting ZSET to the admitted ZSET.
// Returns the customerIDs of admitted customers so the service layer can
// generate tokens and notify them via WebSocket.
func (s *QueueStore) AdmitNextBatch(ctx context.Context, eventID string, n int64) ([]string, error) {
	keys := []string{
		s.waitingKey(eventID),
		s.admittedKey(eventID),
	}
	args := []any{
		n,
		time.Now().UnixNano(),
	}

	res, err := admitBatchScript.Run(ctx, s.client.rdb, keys, args...).StringSlice()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("admitting batch: %w", err)
	}

	return res, nil
}

// GetPosition returns the zero-based position of a customer in the waiting queue.
// Returns -1 if the customer is not in the waiting queue.
func (s *QueueStore) GetPosition(ctx context.Context, eventID, customerID string) (int64, error) {
	pos, err := s.client.rdb.ZRank(ctx, s.waitingKey(eventID), customerID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return -1, nil
		}
		return 0, fmt.Errorf("getting queue position: %w", err)
	}
	return pos, nil
}

// IsAdmitted returns true if the customer is currently in the admitted ZSET.
func (s *QueueStore) IsAdmitted(ctx context.Context, eventID, customerID string) (bool, error) {
	_, err := s.client.rdb.ZScore(ctx, s.admittedKey(eventID), customerID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("checking admission status: %w", err)
	}
	return true, nil
}

// RemoveAdmitted removes a customer from the admitted ZSET.
// Called when a customer successfully claims or their admission window expires.
func (s *QueueStore) RemoveAdmitted(ctx context.Context, eventID, customerID string) error {
	if err := s.client.rdb.ZRem(ctx, s.admittedKey(eventID), customerID).Err(); err != nil {
		return fmt.Errorf("removing from admitted: %w", err)
	}
	return nil
}

// RemoveWaiting removes a customer from the waiting ZSET.
// Called when a customer explicitly abandons the queue.
func (s *QueueStore) RemoveWaiting(ctx context.Context, eventID, customerID string) error {
	if err := s.client.rdb.ZRem(ctx, s.waitingKey(eventID), customerID).Err(); err != nil {
		return fmt.Errorf("removing from waiting: %w", err)
	}
	return nil
}

// EvictExpiredAdmitted atomically finds and removes customers whose
// admission window has expired. Returns their customerIDs so the
// worker can update Postgres and notify them.
func (s *QueueStore) EvictExpiredAdmitted(ctx context.Context, eventID string, admissionTTL time.Duration) ([]string, error) {
	cutoff := fmt.Sprintf("%d", time.Now().Add(-admissionTTL).UnixNano())

	res, err := evictExpiredScript.Run(
		ctx,
		s.client.rdb,
		[]string{s.admittedKey(eventID)},
		cutoff,
	).StringSlice()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("evicting expired admitted customers: %w", err)
	}

	return res, nil
}

// WaitingCount returns the number of customers currently waiting.
// Used by the admission worker to decide batch size.
func (s *QueueStore) WaitingCount(ctx context.Context, eventID string) (int64, error) {
	count, err := s.client.rdb.ZCard(ctx, s.waitingKey(eventID)).Result()
	if err != nil {
		return 0, fmt.Errorf("getting waiting count: %w", err)
	}
	return count, nil
}
