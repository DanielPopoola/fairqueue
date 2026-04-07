package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
)

// InventoryCoordinator wraps the lock and inventory cache into
// a single coherent interface for claim operations.
type InventoryCoordinator struct {
	inventory *redisstore.InventoryStore
	lock      *redisstore.LockStore
	logger    *slog.Logger
}

func NewInventoryCoordinator(
	inventory *redisstore.InventoryStore,
	lock *redisstore.LockStore,
	logger *slog.Logger,
) *InventoryCoordinator {
	return &InventoryCoordinator{
		inventory: inventory,
		lock:      lock,
		logger:    logger,
	}
}

// Acquire attempts to acquire a lock for the given key.
// If Redis is down, logs a warning and returns nil —
// Postgres unique constraint is the safety net.
func (c *InventoryCoordinator) Acquire(ctx context.Context, key string) (acquired bool, err error) {
	err = c.lock.Acquire(ctx, key)
	if err != nil {
		if errors.Is(err, redisstore.ErrLockNotAcquired) {
			return false, redisstore.ErrLockNotAcquired
		}
		// Redis down — proceed without lock
		c.logger.Warn("redis lock unavailable, proceeding without lock",
			"key", key,
			"error", err,
		)
		return false, nil
	}
	return true, nil
}

// Release releases the lock for the given key.
// Only called if the lock was actually acquired.
func (c *InventoryCoordinator) Release(ctx context.Context, key string) {
	if err := c.lock.Release(ctx, key); err != nil {
		c.logger.Warn("failed to release lock",
			"key", key,
			"error", err,
		)
	}
}

// AttemptDecrement atomically checks and decrements inventory.
// Returns whether the decrement succeeded and whether it was
// a cache miss requiring Postgres fallback.
func (c *InventoryCoordinator) AttemptDecrement(ctx context.Context, eventID string) (newCount int64, cacheMiss bool, err error) {
	res, err := c.inventory.DecrementIfAvailable(ctx, eventID)
	if err != nil {
		return 0, false, fmt.Errorf("attempting decrement: %w", err)
	}

	switch {
	case res >= 0:
		return res, false, nil // new count returned
	case res == -1:
		return 0, true, nil // cache miss
	case res == -2:
		return 0, false, fmt.Errorf("sold out: %w", domain.ErrEventSoldOut)
	default:
		return 0, false, fmt.Errorf("unexpected result: %d", res)
	}
}

// Rollback increments the inventory count back after a failed
// Postgres insert. Compensates for the atomic decrement.
func (c *InventoryCoordinator) Rollback(ctx context.Context, eventID string) {
	if err := c.Increment(ctx, eventID); err != nil {
		c.logger.Warn("failed to rollback inventory decrement, reconciliation will heal",
			"event_id", eventID,
			"error", err,
		)
	}
}

// Increment increments the inventory count after a claim is released.
func (c *InventoryCoordinator) Increment(ctx context.Context, eventID string) error {
	_, err := c.inventory.Increment(ctx, eventID)
	if err != nil {
		c.logger.Warn("failed to increment inventory cache, reconciliation will heal",
			"event_id", eventID,
			"error", err,
		)
		return err
	}
	return nil
}

// ForceSync overwrites the Redis count with the authoritative
// Postgres count. Used by the reconciliation worker.
func (c *InventoryCoordinator) ForceSync(ctx context.Context, eventID string, count int64) error {
	if err := c.inventory.ForceSet(ctx, eventID, count); err != nil {
		return fmt.Errorf("syncing inventory count: %w", err)
	}
	return nil
}
