package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
)

// QueueCoordinator wraps Postgres and Redis queue operations into
// a single coherent interface.
type QueueCoordinator struct {
	pgQueue    *postgres.QueueStore
	redisQueue *redisstore.QueueStore
	logger     *slog.Logger
}

func NewQueueCoordinator(
	pgQueue *postgres.QueueStore,
	redisQueue *redisstore.QueueStore,
	logger *slog.Logger,
) *QueueCoordinator {
	return &QueueCoordinator{
		pgQueue:    pgQueue,
		redisQueue: redisQueue,
		logger:     logger,
	}
}

// Join adds a customer to the queue for an event.
// Writes to both Postgres (durable) and Redis (fast position tracking).
func (c *QueueCoordinator) Join(ctx context.Context, entry *domain.QueueEntry) error {
	now := time.Now().UTC()
	entry.JoinedAt = now
	entry.UpdatedAt = now
	if err := c.pgQueue.Create(ctx, entry); err != nil {
		return fmt.Errorf("persisting queue entry: %w", err)
	}

	joinedAt := entry.JoinedAt.UnixNano()

	// Redis second — fast position tracking
	if err := c.redisQueue.Join(ctx, entry.EventID, entry.CustomerID, joinedAt); err != nil {
		// Non-fatal — position tracking is best effort.
		// Customer is in the queue per Postgres.
		c.logger.Warn("failed to add customer to redis queue",
			"customer_id", entry.CustomerID,
			"event_id", entry.EventID,
			"error", err,
		)
	}

	return nil
}

// GetPosition returns the customer's current position in the queue.
// Reads from Redis for speed — falls back to Postgres on cache miss.
func (c *QueueCoordinator) GetPosition(ctx context.Context, eventID, customerID string) (int64, error) {
	// Check admitted ZSET first — if they're here, position is 0 (admitted)
	admitted, err := c.redisQueue.IsAdmitted(ctx, eventID, customerID)
	if err == nil && admitted {
		return 0, nil
	}

	pos, err := c.redisQueue.GetPosition(ctx, eventID, customerID)
	if err != nil {
		return 0, fmt.Errorf("getting position from redis: %w", err)
	}

	if pos == -1 {
		// Redis miss — customer may be in Postgres but not Redis.
		c.logger.Warn("queue position not found in redis, checking postgres",
			"customer_id", customerID,
			"event_id", eventID,
		)
		return c.pgQueue.GetCustomerPosition(ctx, eventID, customerID)
	}

	return pos + 1, nil // convert zero-based to one-based for display
}

// Complete marks a customer's queue entry as completed after
// a successful claim. Updates both Postgres and Redis.
func (c *QueueCoordinator) Complete(ctx context.Context, eventID, customerID string) error {
	entry, err := c.pgQueue.GetByCustomerAndEvent(ctx, customerID, eventID)
	if err != nil {
		return fmt.Errorf("getting queue entry: %w", err)
	}

	if err := c.pgQueue.UpdateStatus(
		ctx,
		entry.ID,
		domain.QueueEntryStatusCompleted,
		domain.QueueEntryStatusAdmitted,
	); err != nil {
		return fmt.Errorf("completing queue entry in postgres: %w", err)
	}

	// Remove from Redis admitted ZSET — non-fatal if it fails.
	// Eviction worker cleans up stragglers.
	if err := c.redisQueue.RemoveAdmitted(ctx, eventID, customerID); err != nil {
		c.logger.Warn("failed to remove customer from redis admitted set",
			"customer_id", customerID,
			"event_id", eventID,
			"error", err,
		)
	}

	return nil
}

// Abandon removes a customer from the queue explicitly.
func (c *QueueCoordinator) Abandon(ctx context.Context, eventID, customerID string) error {
	entry, err := c.pgQueue.GetByCustomerAndEvent(ctx, customerID, eventID)
	if err != nil {
		return fmt.Errorf("getting queue entry: %w", err)
	}

	if err := c.pgQueue.UpdateStatus(
		ctx,
		entry.ID,
		domain.QueueEntryStatusAbandoned,
		domain.QueueEntryStatusWaiting,
	); err != nil {
		return fmt.Errorf("abandoning queue entry: %w", err)
	}

	if err := c.redisQueue.RemoveWaiting(ctx, eventID, customerID); err != nil {
		c.logger.Warn("failed to remove customer from redis waiting set",
			"customer_id", customerID,
			"event_id", eventID,
			"error", err,
		)
	}

	return nil
}

// AdmitNextBatch admits the next n customers from the waiting queue.
// Returns the customerIDs of admitted customers so the caller can
// generate tokens and notify them.
func (c *QueueCoordinator) AdmitNextBatch(ctx context.Context, eventID string, n int64) ([]string, error) {
	// Redis is authoritative for admission — atomic ZPOPMIN
	customerIDs, err := c.redisQueue.AdmitNextBatch(ctx, eventID, n)
	if err != nil {
		return nil, fmt.Errorf("admitting batch from redis: %w", err)
	}

	if len(customerIDs) == 0 {
		return nil, nil
	}

	if err := c.pgQueue.MarkAdmittedBatch(ctx, eventID, customerIDs); err != nil {
		// Non-fatal — expiry worker reconciles stale WAITING entries
		c.logger.Warn("failed to batch admit queue entries in postgres",
			"event_id", eventID,
			"count", len(customerIDs),
			"error", err,
		)
	}

	return customerIDs, nil
}

// EvictExpired removes customers whose admission window has expired.
// Called by worker.
func (c *QueueCoordinator) EvictExpired(ctx context.Context, eventID string, admissionTTL time.Duration) ([]string, error) {
	evicted, err := c.redisQueue.EvictExpiredAdmitted(ctx, eventID, admissionTTL)
	if err != nil {
		return nil, fmt.Errorf("evicting expired from redis: %w", err)
	}

	if len(evicted) == 0 {
		return nil, nil
	}

	if err := c.pgQueue.MarkExpiredBatch(ctx, eventID, evicted); err != nil {
		c.logger.Warn("failed to batch evict admitted queue entries in postgres",
			"event_id", eventID,
			"count", len(evicted),
			"error", err,
		)
	}

	return evicted, nil
}
