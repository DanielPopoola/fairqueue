package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/config"
	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/metrics"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
)

type Notifier interface {
	NotifyAdmitted(ctx context.Context, customerID, token string, logger *slog.Logger) bool
}

// AdmissionWorker runs on every tick for each ACTIVE event:
//  1. Evict admitted customers whose window has expired.
//  2. Calculate how many to admit — capped at available inventory so we
//     never admit more people than there are tickets.
//  3. Admit the next batch, generate a short-lived token per customer,
//     and notify them (stubbed until WebSocket layer exists in Phase 4).
type AdmissionWorker struct {
	events    *postgres.EventStore
	queue     *service.QueueCoordinator
	inventory *service.InventoryCoordinator
	tokenizer *auth.Tokenizer
	notifier  Notifier
	cfg       config.AdmissionWorkerConfig
	logger    *slog.Logger
}

func NewAdmissionWorker(
	events *postgres.EventStore,
	queue *service.QueueCoordinator,
	inventory *service.InventoryCoordinator,
	tokenizer *auth.Tokenizer,
	notifier Notifier,
	cfg config.AdmissionWorkerConfig,
	logger *slog.Logger,
) *AdmissionWorker {
	return &AdmissionWorker{
		events:    events,
		queue:     queue,
		inventory: inventory,
		tokenizer: tokenizer,
		notifier:  notifier,
		cfg:       cfg,
		logger:    logger,
	}
}

func (w *AdmissionWorker) Run(ctx context.Context) error {
	start := time.Now()
	defer func() {
		metrics.WorkerTickDuration.WithLabelValues("admission").Observe(time.Since(start).Seconds())
	}()
	events, err := w.events.GetByStatus(ctx, domain.EventStatusActive)
	if err != nil {
		return fmt.Errorf("fetching active events: %w", err)
	}

	for i := range events {
		event := &events[i]
		if err := w.processEvent(ctx, event); err != nil {
			// Log per-event errors and continue — one bad event must not
			// block admission for all other events.
			w.logger.Error("admission tick failed",
				"event_id", event.ID,
				"error", err,
			)
		}
	}
	return nil
}

func (w *AdmissionWorker) processEvent(ctx context.Context, event *domain.Event) error {
	evicted, err := w.queue.EvictExpired(ctx, event.ID, domain.AdmissionWindowTTL)
	if err != nil {
		// Non-fatal — stale entries don't block admission.
		w.logger.Warn("eviction failed", "event_id", event.ID, "error", err)
	} else if len(evicted) > 0 {
		w.logger.Info("evicted expired admissions",
			"event_id", event.ID,
			"count", len(evicted),
		)
	}

	batchSize, err := w.calculateBatchSize(ctx, event)
	if err != nil {
		return fmt.Errorf("calculating batch size: %w", err)
	}
	if batchSize == 0 {
		return nil // nothing to admit
	}

	admitted, err := w.queue.AdmitNextBatch(ctx, event.ID, batchSize)
	if err != nil {
		return fmt.Errorf("admitting batch: %w", err)
	}
	if len(admitted) == 0 {
		return nil // queue was empty
	}

	w.logger.Info("admitted batch",
		"event_id", event.ID,
		"count", len(admitted),
	)

	return w.notifyAdmitted(ctx, event.ID, admitted)
}

// calculateBatchSize returns how many customers to admit this tick.
// It is capped at available inventory
func (w *AdmissionWorker) calculateBatchSize(ctx context.Context, event *domain.Event) (int64, error) {
	available, err := w.inventory.GetCount(ctx, event.ID)
	if err != nil {
		return 0, fmt.Errorf("reading inventory: %w", err)
	}

	// Cache miss (-1) or sold out (0): admit nothing.
	if available <= 0 {
		return 0, nil
	}

	batchSize := min(available, int64(w.cfg.BatchSize))
	return batchSize, nil
}

// notifyAdmitted generates a signed admission token for each admitted
// customer and sends it to them.
func (w *AdmissionWorker) notifyAdmitted(ctx context.Context, eventID string, customerIDs []string) error {
	metrics.QueueAdmittedTotal.WithLabelValues(eventID).Add(float64(len(customerIDs)))
	for _, customerID := range customerIDs {
		token, err := w.tokenizer.Generate(customerID, eventID)
		if err != nil {
			w.logger.Error("failed to generate admission token",
				"customer_id", customerID,
				"event_id", eventID,
				"error", err,
			)
			continue // one bad token must not block others
		}

		w.notifier.NotifyAdmitted(ctx, customerID, token, w.logger)
	}
	return nil
}
