package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	"github.com/google/uuid"
)

type QueueService struct {
	events    *postgres.EventStore
	customers *postgres.CustomerStore
	queue     *QueueCoordinator
	logger    *slog.Logger
}

func NewQueueService(
	events *postgres.EventStore,
	customers *postgres.CustomerStore,
	queue *QueueCoordinator,
	logger *slog.Logger,
) *QueueService {
	return &QueueService{
		events:    events,
		customers: customers,
		queue:     queue,
		logger:    logger,
	}
}

// JoinResult is returned on successful queue join.
type JoinResult struct {
	QueueEntry *domain.QueueEntry
	Position   int64
}

// Join adds a customer to the waiting queue for an event.
func (s *QueueService) Join(ctx context.Context, customerID, eventID string) (*JoinResult, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("loading event: %w", err)
	}

	if !event.CanAcceptClaims() {
		return nil, domain.ErrEventNotActive
	}

	// JoinedAt is set by QueueCoordinator.Join to ensure
	// Postgres and Redis use the exact same timestamp as score.
	entry := &domain.QueueEntry{
		ID:         uuid.New().String(),
		EventID:    eventID,
		CustomerID: customerID,
		Status:     domain.QueueEntryStatusWaiting,
	}

	if err := s.queue.Join(ctx, entry); err != nil {
		if errors.Is(err, domain.ErrAlreadyInQueue) {
			return nil, domain.ErrAlreadyInQueue
		}
		return nil, fmt.Errorf("joining queue: %w", err)
	}

	position, err := s.queue.GetPosition(ctx, eventID, customerID)
	if err != nil {
		// Non-critical — customer is in queue, position is best effort
		s.logger.Warn("failed to get queue position after join",
			"customer_id", customerID,
			"event_id", eventID,
			"error", err,
		)
		position = 0
	}

	return &JoinResult{
		QueueEntry: entry,
		Position:   position,
	}, nil
}

// GetPosition returns the customer's current position in the queue.
func (s *QueueService) GetPosition(ctx context.Context, customerID, eventID string) (int64, error) {
	position, err := s.queue.GetPosition(ctx, eventID, customerID)
	if err != nil {
		if errors.Is(err, domain.ErrQueueEntryNotFound) {
			return 0, domain.ErrQueueEntryNotFound
		}
		return 0, fmt.Errorf("getting position: %w", err)
	}
	return position, nil
}

func (s *QueueService) GetAdmittedEntry(ctx context.Context, customerID, eventID string) (*domain.QueueEntry, error) {
	return s.queue.pgQueue.GetByCustomerAndEvent(ctx, customerID, eventID)
}

// Abandon removes a customer from the waiting queue.
// Only WAITING entries can be abandoned — ADMITTED entries
// can only move to EXPIRED via the eviction worker.
func (s *QueueService) Abandon(ctx context.Context, customerID, eventID string) error {
	entry, err := s.queue.pgQueue.GetByCustomerAndEvent(ctx, customerID, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrQueueEntryNotFound) {
			return domain.ErrQueueEntryNotFound
		}
		return fmt.Errorf("loading queue entry: %w", err)
	}

	// Domain enforces that only WAITING entries can be abandoned
	if err := entry.Abandon(); err != nil {
		return err // returns ErrInvalidTransition if ADMITTED
	}

	return s.queue.Abandon(ctx, eventID, customerID)
}
