package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DanielPopoola/fairqueue/internal/config"
	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
)

// ReconciliationWorker heals divergence between Redis and Postgres on every tick.
// It also handles stale payment recovery by delegating to PaymentService.Reconcile.
//
// Two jobs per tick:
//  1. Derive authoritative inventory from Postgres, compare with Redis, force-sync
//     if diverged. This is also what heals Redis after a restart.
//  2. Find stale INITIALIZING/PENDING payments and retry or poll Paystack.
type ReconciliationWorker struct {
	events    *postgres.EventStore
	claims    *postgres.ClaimStore
	payments  *service.PaymentService
	inventory *service.InventoryCoordinator
	cfg       config.ReconciliationWorkerConfig
	logger    *slog.Logger
}

func NewReconciliationWorker(
	events *postgres.EventStore,
	claims *postgres.ClaimStore,
	payments *service.PaymentService,
	inventory *service.InventoryCoordinator,
	cfg config.ReconciliationWorkerConfig,
	logger *slog.Logger,
) *ReconciliationWorker {
	return &ReconciliationWorker{
		events:    events,
		claims:    claims,
		payments:  payments,
		inventory: inventory,
		cfg:       cfg,
		logger:    logger,
	}
}

func (w *ReconciliationWorker) Run(ctx context.Context) error {
	if err := w.reconcileInventory(ctx); err != nil {
		// Log and continue — payment reconciliation must still run
		w.logger.Error("inventory reconciliation failed", "error", err)
	}

	if w.payments != nil {
		return w.payments.Reconcile(ctx, w.cfg.StalePaymentAge)
	}

	return nil
}

// reconcileInventory derives the authoritative inventory count from Postgres
// for every ACTIVE and SOLD_OUT event, then force-syncs Redis if the values
// diverge. Running this on every tick means Redis heals within one interval
// after a restart or any write failure.
func (w *ReconciliationWorker) reconcileInventory(ctx context.Context) error {
	// SOLD_OUT events still need reconciliation — a Redis wipe would show
	// them as having inventory again, which would allow incorrect claims.
	for _, status := range []domain.EventStatus{domain.EventStatusActive, domain.EventStatusSoldOut} {
		events, err := w.events.GetByStatus(ctx, status)
		if err != nil {
			return fmt.Errorf("fetching %s events: %w", status, err)
		}
		for i := range events {
			event := &events[i]
			if err := w.reconcileEvent(ctx, event); err != nil {
				// Log per-event errors and continue — one bad event must not
				// block reconciliation for all others.
				w.logger.Error("event inventory reconciliation failed",
					"event_id", event.ID,
					"error", err,
				)
			}
		}
	}
	return nil
}

func (w *ReconciliationWorker) reconcileEvent(ctx context.Context, event *domain.Event) error {
	// Authoritative count: total minus all active (CLAIMED) claims.
	// CONFIRMED claims are already sold — they don't count against available.
	activeClaims, err := w.claims.CountActive(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("counting active claims: %w", err)
	}
	pgCount := int64(event.TotalInventory) - activeClaims

	redisCount, err := w.inventory.GetCount(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("reading redis inventory: %w", err)
	}

	// -1 means the key doesn't exist in Redis (cache miss / restart wipe).
	// Any divergence triggers a force-sync.
	if redisCount == pgCount {
		return nil
	}

	w.logger.Warn("inventory divergence detected, healing",
		"event_id", event.ID,
		"postgres_count", pgCount,
		"redis_count", redisCount,
	)

	return w.inventory.ForceSync(ctx, event.ID, pgCount)
}
