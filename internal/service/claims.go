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
	customerID, err := s.verifyToken(token, eventID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureNotClaimed(ctx, customerID, eventID); err != nil {
		return nil, err
	}

	newCount, err := s.decrementInventory(ctx, customerID, eventID)
	if err != nil {
		return nil, err
	}

	claim, err := s.createClaim(ctx, customerID, eventID)
	if err != nil {
		return nil, err
	}

	if newCount <= 0 {
		if err := s.markEventSoldOut(ctx, eventID, newCount); err != nil {
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

// Release explicitly releases a claim before the TTL expires.
// Restores the Redis inventory count so the next person in queue
// can claim. Postgres is always written first.
//
// Ownership is enforced by the handler before calling this method —
// the handler verifies claim.CustomerID == customerID from context.
func (s *ClaimService) Release(ctx context.Context, claimID string) error {
	claim, err := s.claims.GetByID(ctx, claimID)
	if err != nil {
		return err
	}

	if err := claim.Release(); err != nil {
		return err // ErrClaimNotClaimable if already confirmed or released
	}

	if err := s.claims.UpdateStatus(
		ctx,
		claim.ID,
		domain.ClaimStatusReleased,
		domain.ClaimStatusClaimed,
	); err != nil {
		return fmt.Errorf("releasing claim: %w", err)
	}

	// Restore inventory — non-fatal, reconciliation heals any divergence.
	if err := s.inventory.Increment(ctx, claim.EventID); err != nil {
		s.logger.Warn("failed to restore inventory after explicit release",
			"claim_id", claim.ID,
			"event_id", claim.EventID,
			"error", err,
		)
	}

	s.logger.Info("claim released",
		"claim_id", claim.ID,
		"event_id", claim.EventID,
	)
	return nil
}

func (s *ClaimService) VerifyAdmissionToken(token string) (customerID, eventID string, err error) {
	return s.tokenizer.Verify(token)
}

func (s *ClaimService) verifyToken(token, eventID string) (string, error) {
	customerID, tokenEventID, err := s.tokenizer.Verify(token)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			return "", fmt.Errorf("admission token expired: %w", err)
		}
		return "", fmt.Errorf("invalid admission token: %w", err)
	}
	if tokenEventID != eventID {
		return "", fmt.Errorf("token not valid for this event: %w", auth.ErrTokenInvalid)
	}
	return customerID, nil
}

func (s *ClaimService) ensureNotClaimed(ctx context.Context, customerID, eventID string) error {
	_, err := s.claims.GetByCustomerAndEvent(ctx, customerID, eventID)
	if err == nil {
		return domain.ErrAlreadyClaimed
	}
	if !errors.Is(err, domain.ErrClaimNotFound) {
		return fmt.Errorf("checking existing claim: %w", err)
	}
	return nil
}

func (s *ClaimService) decrementInventory(ctx context.Context, customerID, eventID string) (int64, error) {
	lockKey := customerID + ":" + eventID
	acquired, err := s.inventory.Acquire(ctx, lockKey)
	if err != nil {
		if errors.Is(err, redisstore.ErrLockNotAcquired) {
			return 0, fmt.Errorf("claim already in progress: %w", domain.ErrAlreadyClaimed)
		}
		return 0, fmt.Errorf("acquiring lock: %w", err)
	}
	if acquired {
		defer s.inventory.Release(ctx, lockKey)
	}

	newCount, cacheMiss, err := s.inventory.AttemptDecrement(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrEventSoldOut) {
			return 0, domain.ErrEventSoldOut
		}
		return 0, fmt.Errorf("checking inventory: %w", err)
	}

	if cacheMiss {
		s.logger.Warn("inventory cache miss, falling back to Postgres",
			"event_id", eventID,
		)
		count, err := s.countFromPostgres(ctx, eventID)
		if err != nil {
			return 0, fmt.Errorf("counting inventory from postgres: %w", err)
		}
		if count <= 0 {
			return 0, domain.ErrEventSoldOut
		}
		if err := s.inventory.ForceSync(ctx, eventID, count); err != nil {
			s.logger.Warn("failed to warm inventory cache",
				"event_id", eventID,
				"error", err,
			)
		}
		newCount, _, err = s.inventory.AttemptDecrement(ctx, eventID)
		if err != nil {
			return 0, fmt.Errorf("retrying inventory decrement: %w", err)
		}
	}

	return newCount, nil
}

func (s *ClaimService) createClaim(ctx context.Context, customerID, eventID string) (*domain.Claim, error) {
	claim := &domain.Claim{
		ID:         uuid.NewString(),
		EventID:    eventID,
		CustomerID: customerID,
		Status:     domain.ClaimStatusClaimed,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.claims.Create(ctx, claim); err != nil {
		s.inventory.Rollback(ctx, eventID)
		if postgres.IsUniqueViolation(err) {
			return nil, domain.ErrAlreadyClaimed
		}
		return nil, fmt.Errorf("creating claim: %w", err)
	}

	return claim, nil
}

func (s *ClaimService) markEventSoldOut(ctx context.Context, eventID string, count int64) error {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	event.AvailableInventory = int(count)
	if err := event.OnInventoryDepleted(); err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
		return err
	}

	return s.events.UpdateStatus(ctx, eventID, domain.EventStatusSoldOut)
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
