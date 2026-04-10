package worker_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/config"
	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
	"github.com/DanielPopoola/fairqueue/internal/worker"
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

// ── helpers ──────────────────────────────────────────────────

func buildDeps(t *testing.T) (
	*postgres.EventStore,
	*postgres.ClaimStore,
	*postgres.QueueStore,
	*service.InventoryCoordinator,
	*service.QueueCoordinator,
	*redisstore.QueueStore,
	*redisstore.InventoryStore,
) {
	t.Helper()
	db := postgres.NewDB(testPool)
	rc, err := redisstore.NewClient(testCtx, testClient)
	if err != nil {
		t.Fatalf("redis client: %v", err)
	}
	logger := testhelpers.NewTestLogger()

	invStore := redisstore.NewInventoryStore(rc)
	lockStore := redisstore.NewLockStore(rc, 30*time.Second)
	queueStore := redisstore.NewQueueStore(rc)
	pgQueue := postgres.NewQueueStore(db)

	inv := service.NewInventoryCoordinator(invStore, lockStore, logger)
	qCoord := service.NewQueueCoordinator(pgQueue, queueStore, logger)

	return postgres.NewEventStore(db), postgres.NewClaimStore(db), pgQueue, inv, qCoord, queueStore, invStore
}

// seedWaitingCustomers seeds n customers each with a WAITING queue entry
// for the given event and adds them to the Redis ZSET.
func seedWaitingCustomers(ctx context.Context, t *testing.T, eventID string, n int) []string {
	t.Helper()
	db := postgres.NewDB(testPool)
	rc, _ := redisstore.NewClient(ctx, testClient)
	queueStore := redisstore.NewQueueStore(rc)
	pgQueue := postgres.NewQueueStore(db)

	ids := make([]string, n)
	for i := range n {
		customer := testhelpers.SeedCustomer(ctx, t, testPool)
		ids[i] = customer.ID

		now := time.Now()
		entry := &domain.QueueEntry{
			ID:         uuid.NewString(),
			EventID:    eventID,
			CustomerID: customer.ID,
			Status:     domain.QueueEntryStatusWaiting,
			JoinedAt:   now,
			UpdatedAt:  now,
		}
		if err := pgQueue.Create(ctx, entry); err != nil {
			t.Fatalf("seeding queue entry: %v", err)
		}
		if err := queueStore.Join(ctx, eventID, customer.ID, now.UnixNano()); err != nil {
			t.Fatalf("adding to redis queue: %v", err)
		}
	}
	return ids
}

// ── Reconciliation Worker ─────────────────────────────────────

func TestReconciliationWorker_HealsRedisDivergence(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	eventStores, claimStore, _, inv, _, _, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// Postgres truth: 100 total, 3 active claims = 97 available
	for range 3 {
		customer := testhelpers.SeedCustomer(testCtx, t, testPool)
		testhelpers.SeedClaim(testCtx, t, testPool, customer.ID, event.ID)
	}

	// Redis shows wrong value (diverged after a partial failure)
	invStore.ForceSet(testCtx, event.ID, 50)

	w := worker.NewReconciliationWorker(
		eventStores, claimStore, nil, inv,
		config.ReconciliationWorkerConfig{StalePaymentAge: 5 * time.Minute},
		testhelpers.NewTestLogger(),
	)

	if err := w.Run(testCtx); err != nil {
		t.Fatalf("reconciliation run: %v", err)
	}

	count, err := invStore.Get(testCtx, event.ID)
	if err != nil {
		t.Fatalf("reading redis: %v", err)
	}
	if count != 97 {
		t.Fatalf("expected Redis count 97 after reconciliation, got %d", count)
	}
}

func TestReconciliationWorker_HealsRedisAfterWipe(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	eventStores, claimStore, _, inv, _, _, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// No Redis key at all — simulates a wipe
	w := worker.NewReconciliationWorker(
		eventStores, claimStore, nil, inv,
		config.ReconciliationWorkerConfig{StalePaymentAge: 5 * time.Minute},
		testhelpers.NewTestLogger(),
	)

	if err := w.Run(testCtx); err != nil {
		t.Fatalf("reconciliation run: %v", err)
	}

	count, err := invStore.Get(testCtx, event.ID)
	if err != nil {
		t.Fatalf("reading redis: %v", err)
	}
	if count != int64(event.TotalInventory) {
		t.Fatalf("expected full inventory %d, got %d", event.TotalInventory, count)
	}
}

// ── Expiry Worker ─────────────────────────────────────────────

func TestExpiryWorker_ExpiredClaim_ReleasedAndInventoryRestored(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	_, claimStore, pgQueue, inv, _, _, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	// Prime Redis inventory
	invStore.Set(testCtx, event.ID, 10)

	// Insert a claim backdated past TTL
	_, err := testPool.Exec(testCtx, `
		INSERT INTO claims (id, event_id, customer_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'CLAIMED', $4, $4)`,
		uuid.NewString(), event.ID, customer.ID,
		time.Now().Add(-(domain.ClaimTTL + time.Minute)),
	)
	if err != nil {
		t.Fatalf("seeding expired claim: %v", err)
	}

	w := worker.NewExpiryWorker(
		claimStore, pgQueue, inv,
		config.ExpiryWorkerConfig{},
		testhelpers.NewTestLogger(),
	)

	if err := w.Run(testCtx); err != nil {
		t.Fatalf("expiry run: %v", err)
	}

	// Claim must be RELEASED in Postgres
	expired, err := claimStore.GetExpiredClaims(testCtx)
	if err != nil {
		t.Fatalf("fetching expired: %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("expected 0 expired claims after worker run, got %d", len(expired))
	}

	// Inventory must be restored in Redis
	count, _ := invStore.Get(testCtx, event.ID)
	if count != 11 {
		t.Fatalf("expected inventory 11 after release, got %d", count)
	}
}

func TestExpiryWorker_StaleQueueEntry_MarkedExpired(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	_, _, pgQueue, inv, _, _, _ := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	// Insert a WAITING entry backdated past QueueEntryTTL
	_, err := testPool.Exec(testCtx, `
		INSERT INTO queue_entries (id, event_id, customer_id, status, joined_at, updated_at)
		VALUES ($1, $2, $3, 'WAITING', $4, $4)`,
		uuid.NewString(), event.ID, customer.ID,
		time.Now().Add(-(domain.QueueEntryTTL + time.Minute)),
	)
	if err != nil {
		t.Fatalf("seeding stale entry: %v", err)
	}

	w := worker.NewExpiryWorker(
		postgres.NewClaimStore(postgres.NewDB(testPool)),
		pgQueue, inv,
		config.ExpiryWorkerConfig{},
		testhelpers.NewTestLogger(),
	)

	if err := w.Run(testCtx); err != nil {
		t.Fatalf("expiry run: %v", err)
	}

	// Entry must be EXPIRED — GetActiveByEvent returns only WAITING/ADMITTED
	active, err := pgQueue.GetActiveByEvent(testCtx, event.ID)
	if err != nil {
		t.Fatalf("fetching active entries: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected 0 active queue entries after expiry, got %d", len(active))
	}
}

// ── Admission Worker ──────────────────────────────────────────

func TestAdmissionWorker_AdmitsCorrectBatchSize(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	eventStores, _, _, inv, qCoord, _, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// 50 available tickets, 20 customers waiting
	invStore.Set(testCtx, event.ID, 50)
	seedWaitingCustomers(testCtx, t, event.ID, 20)

	tokenizer := auth.NewTokenizer("test-secret-key-that-is-at-least-32-chars", 5*time.Minute)
	w := worker.NewAdmissionWorker(
		eventStores, qCoord, inv, tokenizer,
		config.AdmissionWorkerConfig{BatchSize: 10},
		testhelpers.NewTestLogger(),
	)

	if err := w.Run(testCtx); err != nil {
		t.Fatalf("admission run: %v", err)
	}

	// Exactly BatchSize customers should have moved to ADMITTED in Postgres
	db := postgres.NewDB(testPool)
	entries, err := postgres.NewQueueStore(db).GetActiveByEvent(testCtx, event.ID)
	if err != nil {
		t.Fatalf("fetching entries: %v", err)
	}

	admitted := 0
	waiting := 0
	for _, e := range entries {
		switch e.Status {
		case domain.QueueEntryStatusAdmitted:
			admitted++
		case domain.QueueEntryStatusWaiting:
			waiting++
		}
	}

	if admitted != 10 {
		t.Fatalf("expected 10 admitted, got %d", admitted)
	}
	if waiting != 10 {
		t.Fatalf("expected 10 still waiting, got %d", waiting)
	}
}

func TestAdmissionWorker_BatchCappedByInventory(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	eventStores, _, _, inv, qCoord, _, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// Only 3 tickets left — batch size of 10 must be capped to 3
	invStore.Set(testCtx, event.ID, 3)
	seedWaitingCustomers(testCtx, t, event.ID, 10)

	tokenizer := auth.NewTokenizer("test-secret-key-that-is-at-least-32-chars", 5*time.Minute)
	w := worker.NewAdmissionWorker(
		eventStores, qCoord, inv, tokenizer,
		config.AdmissionWorkerConfig{BatchSize: 10},
		testhelpers.NewTestLogger(),
	)

	if err := w.Run(testCtx); err != nil {
		t.Fatalf("admission run: %v", err)
	}

	db := postgres.NewDB(testPool)
	entries, _ := postgres.NewQueueStore(db).GetActiveByEvent(testCtx, event.ID)

	admitted := 0
	for _, e := range entries {
		if e.Status == domain.QueueEntryStatusAdmitted {
			admitted++
		}
	}
	if admitted != 3 {
		t.Fatalf("expected 3 admitted (capped by inventory), got %d", admitted)
	}
}

func TestAdmissionWorker_NoAdmissionWhenSoldOut(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	eventStores, _, _, inv, qCoord, _, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// Zero inventory
	invStore.Set(testCtx, event.ID, 0)
	seedWaitingCustomers(testCtx, t, event.ID, 5)

	tokenizer := auth.NewTokenizer("test-secret-key-that-is-at-least-32-chars", 5*time.Minute)
	w := worker.NewAdmissionWorker(
		eventStores, qCoord, inv, tokenizer,
		config.AdmissionWorkerConfig{BatchSize: 10},
		testhelpers.NewTestLogger(),
	)

	if err := w.Run(testCtx); err != nil {
		t.Fatalf("admission run: %v", err)
	}

	db := postgres.NewDB(testPool)
	entries, _ := postgres.NewQueueStore(db).GetActiveByEvent(testCtx, event.ID)
	for _, e := range entries {
		if e.Status == domain.QueueEntryStatusAdmitted {
			t.Fatal("expected no admissions when inventory is zero")
		}
	}
}

// ── Redis Recovery ────────────────────────────────────────────

func TestRecoverRedisState_RebuildsInventoryAndQueue(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	eventStores, claimStore, pgQueue, inv, _, redisQ, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// Seed 3 waiting customers in Postgres (simulating state before wipe)
	for range 3 {
		customer := testhelpers.SeedCustomer(testCtx, t, testPool)
		now := time.Now()
		entry := &domain.QueueEntry{
			ID: uuid.NewString(), EventID: event.ID,
			CustomerID: customer.ID, Status: domain.QueueEntryStatusWaiting,
			JoinedAt: now, UpdatedAt: now,
		}
		pgQueue.Create(testCtx, entry)
		// Deliberately do NOT add to Redis — simulating a wipe
	}

	if err := worker.RecoverRedisState(
		testCtx, eventStores, claimStore, pgQueue, inv, redisQ,
		testhelpers.NewTestLogger(),
	); err != nil {
		t.Fatalf("recovery: %v", err)
	}

	// Inventory cache must be warmed
	count, err := invStore.Get(testCtx, event.ID)
	if err != nil {
		t.Fatalf("reading inventory: %v", err)
	}
	if count != int64(event.TotalInventory) {
		t.Fatalf("expected inventory %d, got %d", event.TotalInventory, count)
	}

	// All 3 WAITING entries must be back in the Redis ZSET
	for i, entry := range func() []domain.QueueEntry {
		e, _ := pgQueue.GetActiveByEvent(testCtx, event.ID)
		return e
	}() {
		pos, err := redisQ.GetPosition(testCtx, event.ID, entry.CustomerID)
		if err != nil {
			t.Fatalf("entry %d not in redis ZSET: %v", i, err)
		}
		if pos < 0 {
			t.Fatalf("entry %d has invalid position %d", i, pos)
		}
	}
}

func TestRecoverRedisState_IsIdempotent(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	eventStores, claimStore, pgQueue, inv, _, redisQ, invStore := buildDeps(t)
	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// Run recovery twice — second run must not overwrite a valid Redis state
	for range 2 {
		if err := worker.RecoverRedisState(
			testCtx, eventStores, claimStore, pgQueue, inv, redisQ,
			testhelpers.NewTestLogger(),
		); err != nil {
			t.Fatalf("recovery: %v", err)
		}
	}

	count, _ := invStore.Get(testCtx, event.ID)
	if count != int64(event.TotalInventory) {
		t.Fatalf("expected inventory unchanged after double recovery, got %d", count)
	}
}
