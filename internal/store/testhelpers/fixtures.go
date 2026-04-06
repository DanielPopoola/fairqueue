package testhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
)

// NewTestDB returns a store.DB backed by the test pool.
func NewTestDB(pool *pgxpool.Pool) *postgres.DB {
	return postgres.NewDB(pool)
}

func SeedOrganizer(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *domain.Organizer {
	t.Helper()

	db := NewTestDB(pool)
	store := postgres.NewOrganizerStore(db)

	organizer := &domain.Organizer{
		ID:           uuid.NewString(),
		Name:         "Test Organizer",
		Email:        uuid.NewString() + "@organizer.com",
		PasswordHash: "$2a$10$fakehashfortest",
		CreatedAt:    time.Now(),
	}

	if err := store.Create(ctx, organizer); err != nil {
		t.Fatalf("seeding organizer: %v", err)
	}

	return organizer
}

func SeedCustomer(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *domain.Customer {
	t.Helper()

	db := NewTestDB(pool)
	store := postgres.NewCustomerStore(db)

	customer := &domain.Customer{
		ID:        uuid.NewString(),
		Email:     uuid.NewString() + "@customer.com",
		CreatedAt: time.Now(),
	}

	if err := store.Create(ctx, customer); err != nil {
		t.Fatalf("seeding customer: %v", err)
	}

	return customer
}

func SeedEvent(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *domain.Event {
	t.Helper()

	organizer := SeedOrganizer(ctx, t, pool)
	db := NewTestDB(pool)
	store := postgres.NewEventStore(db)

	event := &domain.Event{
		ID:             uuid.NewString(),
		OrganizerID:    organizer.ID,
		Name:           "Test Event",
		TotalInventory: 100,
		Price:          500000,
		Status:         domain.EventStatusDraft,
		SaleStart:      time.Now().Add(time.Hour),
		SaleEnd:        time.Now().Add(24 * time.Hour),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := store.Create(ctx, event); err != nil {
		t.Fatalf("seeding event: %v", err)
	}

	return event
}

func SeedClaim(ctx context.Context, t *testing.T, pool *pgxpool.Pool, customerID, eventID string) *domain.Claim {
	t.Helper()

	db := NewTestDB(pool)
	store := postgres.NewClaimStore(db)

	claim := &domain.Claim{
		ID:         uuid.NewString(),
		EventID:    eventID,
		CustomerID: customerID,
		Status:     domain.ClaimStatusClaimed,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := store.Create(ctx, claim); err != nil {
		t.Fatalf("seeding claim: %v", err)
	}

	return claim
}

// SeedActiveEvent inserts a test event in ACTIVE status.
func SeedActiveEvent(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *domain.Event {
	t.Helper()

	event := SeedEvent(ctx, t, pool)
	event.Status = domain.EventStatusActive

	db := NewTestDB(pool)
	store := postgres.NewEventStore(db)

	if err := store.UpdateStatus(ctx, event.ID, domain.EventStatusActive); err != nil {
		t.Fatalf("activating seeded event: %v", err)
	}

	return event
}
