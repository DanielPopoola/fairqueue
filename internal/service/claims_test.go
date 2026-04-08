package service_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
)

var (
	testPool   *pgxpool.Pool
	testClient *redis.Client
	testCtx    = context.Background()
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

	rc, err := testhelpers.NewRedisInstance(testCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setting up redis: %v\n", err)
		os.Exit(1)
	}
	defer rc.Close(testCtx)

	testPool = pg.Pool
	testClient = rc.Client

	os.Exit(m.Run())
}

// buildClaimService wires all dependencies together for testing.
// This is the only place in tests that knows about the full
// dependency graph — mirrors what main.go will do in production.
func buildClaimService(t *testing.T) *service.ClaimService {
	t.Helper()

	db := postgres.NewDB(testPool)
	redisClient, err := redisstore.NewClient(testCtx, testClient)
	if err != nil {
		t.Fatalf("creating redis client: %v", err)
	}

	logger := testhelpers.NewTestLogger()

	inventoryStore := redisstore.NewInventoryStore(redisClient)
	lockStore := redisstore.NewLockStore(redisClient, 30*time.Second)
	redisQueue := redisstore.NewQueueStore(redisClient)
	pgQueue := postgres.NewQueueStore(db)

	inventory := service.NewInventoryCoordinator(inventoryStore, lockStore, logger)
	queue := service.NewQueueCoordinator(pgQueue, redisQueue, logger)

	tokenizer := auth.NewTokenizer("test-secret-key-that-is-at-least-32-chars", 5*time.Minute)

	return service.NewClaimService(
		postgres.NewClaimStore(db),
		postgres.NewEventStore(db),
		inventory,
		queue,
		tokenizer,
		logger,
	)
}

// admitCustomer sets up a customer with a valid admission token
// for the given event. Simulates what the admission worker does.
func admitCustomer(ctx context.Context, t *testing.T, eventID, customerID string) string {
	t.Helper()

	tokenizer := auth.NewTokenizer("test-secret-key-that-is-at-least-32-chars", 5*time.Minute)
	token, err := tokenizer.Generate(customerID, eventID)
	if err != nil {
		t.Fatalf("generating token: %v", err)
	}

	// Add to admitted ZSET in Redis so IsAdmitted check passes
	redisClient, _ := redisstore.NewClient(ctx, testClient)
	redisQueue := redisstore.NewQueueStore(redisClient)

	// Join waiting first then admit
	redisQueue.Join(ctx, eventID, customerID, time.Now().UnixNano())
	redisQueue.AdmitNextBatch(ctx, eventID, 1)

	return token
}

// ─────────────────────────────────────────────────────────────
// Test 1: Concurrency guarantee
// ─────────────────────────────────────────────────────────────

func TestClaimService_Claim_ConcurrencyGuarantee(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildClaimService(t)

	// Set up event with exactly 1 ticket
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// Warm Redis inventory cache with 1
	redisClient, _ := redisstore.NewClient(testCtx, testClient)
	inventory := redisstore.NewInventoryStore(redisClient)
	inventory.Set(testCtx, event.ID, 1)

	// Create 100 customers, each with a valid admission token
	const goroutines = 100
	tokens := make([]string, goroutines)
	customerIDs := make([]string, goroutines)

	for i := range goroutines {
		customer := testhelpers.SeedCustomer(testCtx, t, testPool)
		customerIDs[i] = customer.ID

		// Seed queue entry for each customer
		entry := &domain.QueueEntry{
			ID:         uuid.NewString(),
			EventID:    event.ID,
			CustomerID: customer.ID,
			Status:     domain.QueueEntryStatusAdmitted,
			JoinedAt:   time.Now(),
			AdmittedAt: func() *time.Time { now := time.Now(); return &now }(),
			UpdatedAt:  time.Now(),
		}
		if err := postgres.NewQueueStore(postgres.NewDB(testPool)).Create(testCtx, entry); err != nil {
			t.Fatalf("seeding queue entry: %v", err)
		}

		tokens[i] = admitCustomer(testCtx, t, event.ID, customer.ID)
	}

	// Fire all 100 goroutines simultaneously
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes []string
		failures  int
	)

	for i := range goroutines {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()

			result, err := svc.Claim(testCtx, token, event.ID)

			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				successes = append(successes, result.Claim.ID)
			} else {
				failures++
			}
		}(tokens[i])
	}

	wg.Wait()

	// Exactly 1 should succeed
	if len(successes) != 1 {
		t.Fatalf("expected exactly 1 successful claim, got %d", len(successes))
	}

	if failures != goroutines-1 {
		t.Fatalf("expected %d failures, got %d", goroutines-1, failures)
	}

	// Verify in Postgres — exactly 1 CLAIMED row
	db := postgres.NewDB(testPool)
	store := postgres.NewClaimStore(db)
	count, err := store.CountActive(testCtx, event.ID)
	if err != nil {
		t.Fatalf("counting claims: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 active claim in postgres, got %d", count)
	}
}

// ─────────────────────────────────────────────────────────────
// Test 2: Cache miss falls back to Postgres
// ─────────────────────────────────────────────────────────────

func TestClaimService_Claim_CacheMissFallsBackToPostgres(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildClaimService(t)

	// Set up event with 10 tickets
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	// Deliberately do NOT warm Redis cache —
	// inventory key does not exist in Redis

	// Seed queue entry
	entry := &domain.QueueEntry{
		ID:         uuid.NewString(),
		EventID:    event.ID,
		CustomerID: customer.ID,
		Status:     domain.QueueEntryStatusAdmitted,
		JoinedAt:   time.Now(),
		AdmittedAt: func() *time.Time { now := time.Now(); return &now }(),
		UpdatedAt:  time.Now(),
	}
	postgres.NewQueueStore(postgres.NewDB(testPool)).Create(testCtx, entry)

	token := admitCustomer(testCtx, t, event.ID, customer.ID)

	result, err := svc.Claim(testCtx, token, event.ID)
	if err != nil {
		t.Fatalf("expected claim to succeed via postgres fallback, got: %v", err)
	}
	if result.Claim.CustomerID != customer.ID {
		t.Fatalf("expected claim for customer %s, got %s", customer.ID, result.Claim.CustomerID)
	}
}

// ─────────────────────────────────────────────────────────────
// Test 3: Sold out transition
// ─────────────────────────────────────────────────────────────

func TestClaimService_Claim_TransitionsEventToSoldOut(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildClaimService(t)

	// Event with exactly 1 ticket
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	// Warm Redis with count of 1
	redisClient, _ := redisstore.NewClient(testCtx, testClient)
	inventory := redisstore.NewInventoryStore(redisClient)
	inventory.Set(testCtx, event.ID, 1)

	// Seed queue entry
	entry := &domain.QueueEntry{
		ID:         uuid.NewString(),
		EventID:    event.ID,
		CustomerID: customer.ID,
		Status:     domain.QueueEntryStatusAdmitted,
		JoinedAt:   time.Now(),
		AdmittedAt: func() *time.Time { now := time.Now(); return &now }(),
		UpdatedAt:  time.Now(),
	}
	postgres.NewQueueStore(postgres.NewDB(testPool)).Create(testCtx, entry)

	token := admitCustomer(testCtx, t, event.ID, customer.ID)

	_, err := svc.Claim(testCtx, token, event.ID)
	if err != nil {
		t.Fatalf("expected claim to succeed, got: %v", err)
	}

	// Event should now be SOLD_OUT in Postgres
	db := postgres.NewDB(testPool)
	eventStore := postgres.NewEventStore(db)

	updated, err := eventStore.GetByID(testCtx, event.ID)
	if err != nil {
		t.Fatalf("fetching event: %v", err)
	}
	if updated.Status != domain.EventStatusSoldOut {
		t.Fatalf("expected event status SOLD_OUT, got %s", updated.Status)
	}
}
