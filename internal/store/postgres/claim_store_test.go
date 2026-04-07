package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/store/postgres"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
)

var (
	testPool *pgxpool.Pool
	testCtx  = context.Background()
)

func TestMain(m *testing.M) {
	pg, err := testhelpers.NewPostgresInstance(testCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setting up postgres: %v\n", err)
		os.Exit(1)
	}
	defer pg.Close(testCtx)

	if err := testhelpers.RunMigrationsOnPool(testCtx, pg.Pool); err != nil {
		fmt.Fprintf(os.Stderr, "running migrations: %v\n", err)
		os.Exit(1)
	}

	testPool = pg.Pool
	os.Exit(m.Run())
}

func TestClaimStore_Create_DuplicateActiveClaim(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool) // clean slate

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	// First claim succeeds
	testhelpers.SeedClaim(testCtx, t, testPool, customer.ID, event.ID)

	// Second claim for same customer + event should fail —
	// partial unique index on (customer_id, event_id) WHERE status = 'CLAIMED'
	db := postgres.NewDB(testPool)
	store := postgres.NewClaimStore(db)

	duplicate := &domain.Claim{
		ID:         uuid.NewString(),
		EventID:    event.ID,
		CustomerID: customer.ID,
		Status:     domain.ClaimStatusClaimed,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.Create(testCtx, duplicate)
	if err != domain.ErrAlreadyClaimed {
		t.Fatal("expected error for duplicate active claim, got nil")
	}
}

func TestClaimStore_Create_AllowsClaimAfterRelease(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool) // clean slate

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	db := postgres.NewDB(testPool)
	store := postgres.NewClaimStore(db)

	// First claim
	first := testhelpers.SeedClaim(testCtx, t, testPool, customer.ID, event.ID)

	// Release it
	if err := store.UpdateStatus(testCtx, first.ID, domain.ClaimStatusReleased, domain.ClaimStatusClaimed); err != nil {
		t.Fatalf("releasing claim: %v", err)
	}

	// Second claim for same customer + event should now succeed —
	// partial index ignores RELEASED rows
	second := &domain.Claim{
		ID:         uuid.NewString(),
		EventID:    event.ID,
		CustomerID: customer.ID,
		Status:     domain.ClaimStatusClaimed,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := store.Create(testCtx, second); err != nil {
		t.Fatalf("expected second claim to succeed after release, got: %v", err)
	}
}

func TestClaimStore_UpdateStatus_WrongExpectedStatus(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool) // clean slate

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)
	claim := testhelpers.SeedClaim(testCtx, t, testPool, customer.ID, event.ID)

	db := postgres.NewDB(testPool)
	store := postgres.NewClaimStore(db)

	// Try to confirm a CLAIMED claim but pass wrong expected status.
	// The conditional UPDATE should affect 0 rows.
	err := store.UpdateStatus(testCtx, claim.ID, domain.ClaimStatusConfirmed, domain.ClaimStatusReleased)
	if err != domain.ErrClaimNotFound {
		t.Fatalf("expected ErrClaimNotFound, got: %v", err)
	}

	// Verify claim is unchanged
	fetched, err := store.GetByID(testCtx, claim.ID)
	if err != nil {
		t.Fatalf("fetching claim: %v", err)
	}
	if fetched.Status != domain.ClaimStatusClaimed {
		t.Fatalf("expected status CLAIMED, got: %s", fetched.Status)
	}
}

func TestClaimStore_GetExpiredClaims(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool) // clean slate

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	db := postgres.NewDB(testPool)
	store := postgres.NewClaimStore(db)

	// Insert an expired claim by backdating created_at
	expiredCustomer := testhelpers.SeedCustomer(testCtx, t, testPool)
	_, err := testPool.Exec(testCtx, `
		INSERT INTO claims (id, event_id, customer_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'CLAIMED', $4, $4)`,
		uuid.NewString(),
		event.ID,
		expiredCustomer.ID,
		time.Now().Add(-(domain.ClaimTTL + time.Minute)),
	)
	if err != nil {
		t.Fatalf("inserting backdated claim: %v", err)
	}

	// Insert a fresh claim — should NOT appear in expired results
	freshCustomer := testhelpers.SeedCustomer(testCtx, t, testPool)
	testhelpers.SeedClaim(testCtx, t, testPool, freshCustomer.ID, event.ID)

	expired, err := store.GetExpiredClaims(testCtx)
	if err != nil {
		t.Fatalf("getting expired claims: %v", err)
	}

	if len(expired) != 1 {
		t.Fatalf("expected 1 expired claim, got %d", len(expired))
	}
	if expired[0].CustomerID != expiredCustomer.ID {
		t.Fatalf("expected expired claim for customer %s, got %s", expiredCustomer.ID, expired[0].CustomerID)
	}
}
