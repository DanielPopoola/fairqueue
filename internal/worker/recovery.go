package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
)

// RecoverRedisState runs once at startup before the worker loops begin.
// It handles the case where Redis was wiped (restart, eviction, flush)
// while Postgres retained its state.
//
// For each ACTIVE event:
//  1. Warm the inventory cache using SET NX — if the key already exists
//     (Redis was not wiped), this is a no-op. Safe to always call.
//  2. Re-add all WAITING queue entries to the Redis ZSET using their
//     original joined_at timestamp as the score, preserving FIFO order.
//
// This does not reconstruct ADMITTED entries — customers whose admission
// window survived a Redis wipe will be caught by the expiry worker on
// its first tick and expired from Postgres.
func RecoverRedisState(
	ctx context.Context,
	events *postgres.EventStore,
	claims *postgres.ClaimStore,
	queue *postgres.QueueStore,
	inventory *service.InventoryCoordinator,
	redisQueue *redisstore.QueueStore,
	logger *slog.Logger,
) error {
	logger.Info("starting Redis state recovery")

	activeEvents, err := events.GetByStatus(ctx, domain.EventStatusActive)
	if err != nil {
		return fmt.Errorf("fetching active events for recovery: %w", err)
	}

	for i := range activeEvents {
		event := &activeEvents[i]
		if err := recoverEvent(ctx, event, claims, queue, inventory, redisQueue, logger); err != nil {
			// Log per-event failures and continue — a partial recovery is
			// better than no recovery. The reconciliation worker heals
			// any remaining divergence on its first tick.
			logger.Error("failed to recover event state",
				"event_id", event.ID,
				"error", err,
			)
		}
	}

	logger.Info("Redis state recovery complete", "event_count", len(activeEvents))
	return nil
}

func recoverEvent(
	ctx context.Context,
	event *domain.Event,
	claims *postgres.ClaimStore,
	queue *postgres.QueueStore,
	inventory *service.InventoryCoordinator,
	redisQueue *redisstore.QueueStore,
	logger *slog.Logger,
) error {
	// Derive authoritative inventory count from Postgres.
	activeClaims, err := claims.CountActive(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("counting active claims: %w", err)
	}
	pgCount := int64(event.TotalInventory) - activeClaims

	// SET NX — only sets if the key does not exist.
	set, err := inventory.WarmCache(ctx, event.ID, pgCount)
	if err != nil {
		return fmt.Errorf("warming inventory cache: %w", err)
	}
	if set {
		logger.Info("inventory cache warmed",
			"event_id", event.ID,
			"count", pgCount,
		)
	}

	// Re-add WAITING entries to the Redis ZSET.
	// Only WAITING — admitted entries are handled by the expiry worker.
	waitingEntries, err := queue.GetActiveByEvent(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("fetching waiting entries: %w", err)
	}

	for i := range waitingEntries {
		entry := &waitingEntries[i]
		if entry.Status != domain.QueueEntryStatusWaiting {
			continue
		}
		// Join uses the original joined_at timestamp as score so FIFO
		// order is exactly preserved after recovery.
		if err := redisQueue.Join(ctx, event.ID, entry.CustomerID, entry.JoinedAt.UnixNano()); err != nil {
			if errors.Is(err, domain.ErrAlreadyInQueue) {
				continue // already in Redis — not wiped, no-op
			}
			logger.Warn("failed to re-add queue entry during recovery",
				"entry_id", entry.ID,
				"customer_id", entry.CustomerID,
				"event_id", event.ID,
				"error", err,
			)
		}
	}

	return nil
}
