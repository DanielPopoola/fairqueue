package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/domain"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
)

type ClaimService struct {
	claims    *postgres.ClaimStore
	events    *postgres.EventStore
	inventory *InventoryCoordinator
	queue     *QueueCoordinator
	tokenizer *auth.Tokenizer
	logger    *slog.Logger
}

func NewClaimService(
	claims *postgres.ClaimStore,
	events *postgres.EventStore,
	inventory *InventoryCoordinator,
	queue *QueueCoordinator,
	tokenizer *auth.Tokenizer,
	logger *slog.Logger,
) *ClaimService {
	return &ClaimService{
		claims:    claims,
		events:    events,
		inventory: inventory,
		queue:     queue,
		tokenizer: tokenizer,
		logger:    logger,
	}
}

type ClaimResult struct {
	Claim   *domain.Claim
	EventID string
}

func (s *ClaimService) Claim(ctx context.Context, token, eventID string) (*ClaimResult, error) {
	customerID, tokenEventID, err := s.tokenizer.Verify(token)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			return nil, fmt.Errorf("admission token expired: %w", err)
		}
		return nil, fmt.Errorf("invalid admission token: %w", err)
	}

	if tokenEventID != eventID {
		return nil, fmt.Errorf("token not valid for this event: %w", auth.ErrTokenInvalid)
	}

	_, err = s.claims.GetByCustomerAndEvent(ctx, customerID, eventID)
	if err == nil {
		return nil, domain.ErrAlreadyClaimed
	}
	if !errors.Is(err, domain.ErrClaimNotFound) {
		return nil, fmt.Errorf("invalid admission token: %w", err)
	}

	lockKey := customerID + ":" + eventID
	acquired, err := s.inventory.Acquire(ctx, lockKey)
	if err != nil {
		if errors.Is(err, redisstore.ErrLockNotAcquired) {
			return nil, fmt.Errorf("claim already in progress: %w", domain.ErrAlreadyClaimed)
		}
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}
	if acquired {
		defer s.inventory.Release(ctx, lockKey)
	}

	count, err := s.inventory.GetCount(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("checking inventory: %w", err)
	}

	if count == -1 {
		s.logger.Warn("inventory cache miss, falling back to postgres",
			"event_id", eventID,
		)
		count, err = s.countFromPostgres(ctx, eventID)
		if err != nil {
			return nil, fmt.Errorf("counting inventory from postgres: %w", err)
		}
	}

	if count <= 0 {
		return nil, domain.ErrEventSoldOut
	}

	claim := &domain.Claim{
		ID:         uuid.NewString(),
		EventID:    eventID,
		CustomerID: customerID,
		Status:     domain.ClaimStatusClaimed,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.claims.Create(ctx, claim); err != nil {
		if postgres.IsUniqueViolation(err) {
			return nil, domain.ErrAlreadyClaimed
		}
		return nil, fmt.Errorf("creating claim: %w", err)
	}

	newCount, err := s.inventory.Decrement(ctx, eventID)
	if err != nil {
		s.logger.Warn("failed to decrement inventory, reconciliation will heal",
			"event_id", eventID,
			"claim_id", claim.ID,
		)
	}

	if err == nil && newCount <= 0 {
		if err := s.markEventSoldOutIfDepleted(ctx, eventID, newCount); err != nil {
			s.logger.Warn("failed to mark event sold out",
				"event_id", eventID,
				"error", err,
			)
		}
	}

	if err := s.queue.Complete(ctx, eventID, customerID); err != nil {
		s.logger.Warn("failed to complete queue entry",
			"customer_id", customerID,
			"event_id", eventID,
			"error", err,
		)
	}

	return &ClaimResult{Claim: claim, EventID: eventID}, nil
}

func (s *ClaimService) countFromPostgres(ctx context.Context, eventID string) (int64, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return 0, err
	}

	claimedCount, err := s.claims.CountActive(ctx, eventID)
	if err != nil {
		return 0, err
	}

	return int64(event.TotalInventory) - claimedCount, nil
}

func (s *ClaimService) markEventSoldOutIfDepleted(ctx context.Context, eventID string, count int64) error {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	event.AvailableInventory = int(count)
	if err := event.OnInventoryDepleted(); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			return nil
		}
		return err
	}

	return s.events.UpdateStatus(ctx, eventID, domain.EventStatusSoldOut)
}
