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

// SeedEvent inserts a test event and returns it.
// Callers can override fields by modifying the returned struct
// before using it in assertions.
func SeedEvent(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *domain.Event {
	t.Helper()

	// Create the organizer first so the foreign key is satisfied
	organizerID := SeedUser(ctx, t, pool)

	db := NewTestDB(pool)
	store := postgres.NewEventStore(db)

	event := &domain.Event{
		ID:             uuid.NewString(),
		OrganizerID:    organizerID,
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

// SeedUser inserts a test user and returns their ID.
func SeedUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	userID := uuid.NewString()
	_, err := pool.Exec(ctx,
		"INSERT INTO users (id, email) VALUES ($1, $2)",
		userID,
		userID+"@test.com",
	)
	if err != nil {
		t.Fatalf("seeding user: %v", err)
	}

	return userID
}

// SeedClaim inserts a test claim in CLAIMED status.
func SeedClaim(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID, eventID string) *domain.Claim {
	t.Helper()

	db := NewTestDB(pool)
	store := postgres.NewClaimStore(db)

	claim := &domain.Claim{
		ID:        uuid.NewString(),
		EventID:   eventID,
		UserID:    userID,
		Status:    domain.ClaimStatusClaimed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Create(ctx, claim); err != nil {
		t.Fatalf("seeding claim: %v", err)
	}

	return claim
}
