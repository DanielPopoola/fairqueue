package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DanielPopoola/fairqueue/internal/config"
	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/metrics"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
)

// ExpiryWorker runs two cleanup jobs on every tick:
//  1. Release claims whose 10-minute window has elapsed without payment.
//     Each release increments the Redis inventory count so the ticket
//     returns to the available pool.
//  2. Expire stale queue entries — WAITING entries older than QueueEntryTTL
//     and ADMITTED entries whose admission window has elapsed.
type ExpiryWorker struct {
	claims    *postgres.ClaimStore
	queue     *postgres.QueueStore
	inventory *service.InventoryCoordinator
	cfg       config.ExpiryWorkerConfig
	logger    *slog.Logger
}

func NewExpiryWorker(
	claims *postgres.ClaimStore,
	queue *postgres.QueueStore,
	inventory *service.InventoryCoordinator,
	cfg config.ExpiryWorkerConfig,
	logger *slog.Logger,
) *ExpiryWorker {
	return &ExpiryWorker{
		claims:    claims,
		queue:     queue,
		inventory: inventory,
		cfg:       cfg,
		logger:    logger,
	}
}

func (w *ExpiryWorker) Run(ctx context.Context) error {
	if err := w.expireClaims(ctx); err != nil {
		w.logger.Error("claim expiry failed", "error", err)
	}
	return w.expireQueueEntries(ctx)
}

// expireClaims finds all CLAIMED rows older than ClaimTTL, marks them
// RELEASED in Postgres, then increments the Redis inventory count.
// Postgres is always written first — if the Redis increment fails,
// the reconciliation worker heals the count on its next tick.
func (w *ExpiryWorker) expireClaims(ctx context.Context) error {
	expired, err := w.claims.GetExpiredClaims(ctx)
	if err != nil {
		return fmt.Errorf("fetching expired claims: %w", err)
	}

	for i := range expired {
		claim := &expired[i]
		if err := w.releaseClaim(ctx, claim); err != nil {
			w.logger.Error("failed to release expired claim",
				"claim_id", claim.ID,
				"event_id", claim.EventID,
				"error", err,
			)
			// Continue — one stuck claim must not block the rest.
		}
	}
	return nil
}

func (w *ExpiryWorker) releaseClaim(ctx context.Context, claim *domain.Claim) error {
	// Conditional update — only releases if still CLAIMED.
	// Safe to call multiple times; second call is a no-op.
	if err := w.claims.UpdateStatus(
		ctx,
		claim.ID,
		domain.ClaimStatusReleased,
		domain.ClaimStatusClaimed,
	); err != nil {
		if errors.Is(err, domain.ErrClaimNotFound) {
			// Already released by another path (e.g. payment failure webhook).
			return nil
		}
		return fmt.Errorf("releasing claim: %w", err)
	}

	metrics.ClaimsExpiredTotal.WithLabelValues(claim.EventID).Inc()
	// Postgres committed — now restore inventory in Redis.
	// Non-fatal: reconciliation worker heals any divergence.
	if err := w.inventory.Increment(ctx, claim.EventID); err != nil {
		w.logger.Warn("failed to increment inventory after claim release, reconciliation will heal",
			"claim_id", claim.ID,
			"event_id", claim.EventID,
			"error", err,
		)
	}

	w.logger.Info("expired claim released",
		"claim_id", claim.ID,
		"event_id", claim.EventID,
	)
	return nil
}

// expireQueueEntries finds WAITING entries older than QueueEntryTTL and
// ADMITTED entries older than AdmissionWindowTTL, marks them EXPIRED.
// Redis ZSET cleanup is handled by QueueCoordinator.EvictExpired —
// here we only need the Postgres side since GetStaleEntries reads from Postgres.
func (w *ExpiryWorker) expireQueueEntries(ctx context.Context) error {
	stale, err := w.queue.GetStaleEntries(
		ctx,
		domain.QueueEntryTTL,
		domain.AdmissionWindowTTL,
	)
	if err != nil {
		return fmt.Errorf("fetching stale queue entries: %w", err)
	}

	for i := range stale {
		entry := &stale[i]
		if err := w.queue.UpdateStatus(
			ctx,
			entry.ID,
			domain.QueueEntryStatusExpired,
			entry.Status, // conditional on current status — idempotent
		); err != nil {
			if errors.Is(err, domain.ErrQueueEntryNotFound) {
				continue // already expired by another path
			}
			w.logger.Error("failed to expire queue entry",
				"entry_id", entry.ID,
				"error", err,
			)
		}
	}
	return nil
}
