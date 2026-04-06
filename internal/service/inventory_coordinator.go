package service

import (
	"context"
	"fmt"
	"log/slog"

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
		if err == redisstore.ErrLockNotAcquired {
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

// GetCount returns the cached inventory count for an event.
// Returns -1 on cache miss.
func (c *InventoryCoordinator) GetCount(ctx context.Context, eventID string) (int64, error) {
	return c.inventory.Get(ctx, eventID)
}

// Decrement decrements the inventory count after a successful claim.
// Returns the new count.
func (c *InventoryCoordinator) Decrement(ctx context.Context, eventID string) (int64, error) {
	count, err := c.inventory.Decrement(ctx, eventID)
	if err != nil {
		c.logger.Warn("failed to decrement inventory cache, reconciliation will heal",
			"event_id", eventID,
			"error", err,
		)
		return 0, err
	}
	return count, nil
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
