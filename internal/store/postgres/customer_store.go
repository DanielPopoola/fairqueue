package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/DanielPopoola/fairqueue/internal/domain"
)

type CustomerStore struct {
	exec Executor
}

func NewCustomerStore(db *DB) *CustomerStore {
	return &CustomerStore{exec: db.Pool}
}

func (s *CustomerStore) Create(ctx context.Context, customer *domain.Customer) error {
	query := `
		INSERT INTO customers (id, email, created_at)
		VALUES ($1, $2, $3)`

	_, err := s.exec.Exec(ctx, query,
		customer.ID,
		customer.Email,
		customer.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting customer: %w", err)
	}
	return nil
}

func (s *CustomerStore) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	query := `
		SELECT id, email, created_at
		FROM customers
		WHERE id = $1`

	var c domain.Customer
	err := s.exec.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Email,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("getting customer: %w", err)
	}
	return &c, nil
}

func (s *CustomerStore) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	query := `
		SELECT id, email, created_at
		FROM customers
		WHERE email = $1`

	var c domain.Customer
	err := s.exec.QueryRow(ctx, query, email).Scan(
		&c.ID,
		&c.Email,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("getting customer: %w", err)
	}
	return &c, nil
}

// GetOrCreate finds a customer by email or creates one if they
// don't exist yet. This is the OTP flow entry point — a customer
// is created the first time they request an OTP.
func (s *CustomerStore) GetOrCreate(ctx context.Context, email string) (*domain.Customer, error) {
	existing, err := s.GetByEmail(ctx, email)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		return nil, err
	}

	customer := &domain.Customer{
		ID:        uuid.NewString(),
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := s.Create(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}
