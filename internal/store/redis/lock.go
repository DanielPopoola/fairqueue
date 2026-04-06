package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrLockNotAcquired = errors.New("lock already held")

const lockKeyFmt = "lock:%s" // lock:{claim_id}

// LockStore manages distributed locks for claim operations.
type LockStore struct {
	client *Client
	ttl    time.Duration
}

func NewLockStore(client *Client, ttl time.Duration) *LockStore {
	return &LockStore{client: client, ttl: ttl}
}

func (s *LockStore) lockKey(claimID string) string {
	return fmt.Sprintf(lockKeyFmt, claimID)
}

// Acquire attempts to acquire a lock for the given claim ID.
// Returns ErrLockNotAcquired if the lock is already held.
// The lock auto-expires after the configured TTL — if the server
// crashes mid-claim, the lock is released automatically.
func (s *LockStore) Acquire(ctx context.Context, claimID string) error {
	err := s.client.rdb.SetArgs(ctx, s.lockKey(claimID), "1", redis.SetArgs{
		Mode: "NX",
		TTL:  s.ttl,
	}).Err()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrLockNotAcquired
		}
		return fmt.Errorf("acquiring lock: %w", err)
	}

	return nil
}

// Release deletes the lock for the given claim ID.
// Called after the claim operation completes — success or failure.
func (s *LockStore) Release(ctx context.Context, claimID string) error {
	if err := s.client.rdb.Del(ctx, s.lockKey(claimID)).Err(); err != nil {
		return fmt.Errorf("releasing lock: %w", err)
	}
	return nil
}
