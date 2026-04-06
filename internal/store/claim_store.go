package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/jackc/pgx/v5"
)

type ClaimStore struct {
	db *DB
}

func NewClaimStore(db *DB) *ClaimStore {
	return &ClaimStore{db: db}
}

func (s *ClaimStore) Create(ctx context.Context, claim *domain.Claim) error {
	query := `
		INSERT INTO claims (
			id, event_id, user_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.Pool.Exec(ctx, query,
		claim.ID,
		claim.EventID,
		claim.UserID,
		claim.Status,
		claim.CreatedAt,
		claim.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting claim: %w", err)
	}

	return nil
}

func (s *ClaimStore) GetByID(ctx context.Context, id string) (*domain.Claim, error) {
	query := `
		SELECT id, event_id, user_id, status, created_at, updated_at
		FROM claims
		WHERE id = $1
	`

	var c domain.Claim
	err := s.db.Pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.EventID,
		&c.UserID,
		&c.Status,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrClaimNotFound
		}
		return nil, fmt.Errorf("getting claim: %w", err)
	}

	return &c, nil
}

func (s *ClaimStore) GetByUserAndEvent(ctx context.Context, userID, eventID string) (*domain.Claim, error) {
	query := `
        SELECT id, event_id, user_id, status, created_at, updated_at
        FROM claims
        WHERE user_id = $1
        AND event_id = $2
        AND status = 'CLAIMED'
	`

	var c domain.Claim
	err := s.db.Pool.QueryRow(ctx, query, userID, eventID).Scan(
		&c.ID,
		&c.EventID,
		&c.UserID,
		&c.Status,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrClaimNotFound
		}
		return nil, fmt.Errorf("getting claim by user and event: %w", err)
	}
	return &c, nil
}

func (s *ClaimStore) GetExpiredClaims(ctx context.Context) ([]domain.Claim, error) {
	query := `
		SELECT id, event_id, user_id, status, created_at, updated_at
        FROM claims
        WHERE status = 'CLAIMED'
        AND created_at < $1
	`
	rows, err := s.db.Pool.Query(ctx, query, time.Now().Add(-domain.ClaimTTL))
	if err != nil {
		return nil, fmt.Errorf("querying expired claims: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Claim])
}

func (s *ClaimStore) UpdateStatus(
	ctx context.Context,
	id string,
	newStatus domain.ClaimStatus,
	expectedStatus domain.ClaimStatus,
) error {
	query := `
        UPDATE claims
        SET status = $1, updated_at = $2
        WHERE id = $3
        AND status = $4
	`

	result, err := s.db.Pool.Exec(ctx, query,
		newStatus,
		time.Now(),
		id,
		expectedStatus,
	)
	if err != nil {
		return fmt.Errorf("updating claim status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrClaimNotFound
	}
	return nil
}
