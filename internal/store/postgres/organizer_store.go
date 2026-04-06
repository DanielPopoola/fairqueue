package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/DanielPopoola/fairqueue/internal/domain"
)

type OrganizerStore struct {
	db *DB
}

func NewOrganizerStore(db *DB) *OrganizerStore {
	return &OrganizerStore{db: db}
}

func (s *OrganizerStore) Create(ctx context.Context, organizer *domain.Organizer) error {
	query := `
		INSERT INTO organizers (id, name, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.db.Pool.Exec(ctx, query,
		organizer.ID,
		organizer.Name,
		organizer.Email,
		organizer.PasswordHash,
		organizer.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting organizer: %w", err)
	}
	return nil
}

func (s *OrganizerStore) GetByID(ctx context.Context, id string) (*domain.Organizer, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM organizers
		WHERE id = $1`

	var o domain.Organizer
	err := s.db.Pool.QueryRow(ctx, query, id).Scan(
		&o.ID,
		&o.Name,
		&o.Email,
		&o.PasswordHash,
		&o.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizerNotFound
		}
		return nil, fmt.Errorf("getting organizer: %w", err)
	}
	return &o, nil
}

func (s *OrganizerStore) GetByEmail(ctx context.Context, email string) (*domain.Organizer, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM organizers
		WHERE email = $1`

	var o domain.Organizer
	err := s.db.Pool.QueryRow(ctx, query, email).Scan(
		&o.ID,
		&o.Name,
		&o.Email,
		&o.PasswordHash,
		&o.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizerNotFound
		}
		return nil, fmt.Errorf("getting organizer by email: %w", err)
	}
	return &o, nil
}
