package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
	"github.com/google/uuid"
)

// TestMain is already defined in claims_test.go in the same package.
// testPool and testClient are shared across all service tests.

func buildQueueService(t *testing.T) *service.QueueService {
	t.Helper()

	db := postgres.NewDB(testPool)
	redisClient, err := redisstore.NewClient(testCtx, testClient)
	if err != nil {
		t.Fatalf("creating redis client: %v", err)
	}

	logger := testhelpers.NewTestLogger()

	redisQueue := redisstore.NewQueueStore(redisClient)
	pgQueue := postgres.NewQueueStore(db)
	queue := service.NewQueueCoordinator(pgQueue, redisQueue, logger)

	return service.NewQueueService(
		postgres.NewEventStore(db),
		postgres.NewCustomerStore(db),
		queue,
		logger,
	)
}

func TestQueueService_Join_ActiveEvent(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildQueueService(t)

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	result, err := svc.Join(testCtx, customer.ID, event.ID)
	if err != nil {
		t.Fatalf("expected join to succeed, got: %v", err)
	}

	if result.QueueEntry.CustomerID != customer.ID {
		t.Fatalf("expected customer %s, got %s", customer.ID, result.QueueEntry.CustomerID)
	}

	if result.Position != 1 {
		t.Fatalf("expected position 1, got %d", result.Position)
	}
}

func TestQueueService_Join_InactiveEvent(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildQueueService(t)

	// DRAFT event — not active
	event := testhelpers.SeedEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	_, err := svc.Join(testCtx, customer.ID, event.ID)
	if !errors.Is(err, domain.ErrEventNotActive) {
		t.Fatalf("expected ErrEventNotActive, got: %v", err)
	}
}

func TestQueueService_Join_DuplicateRejected(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildQueueService(t)

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	// First join succeeds
	if _, err := svc.Join(testCtx, customer.ID, event.ID); err != nil {
		t.Fatalf("first join failed: %v", err)
	}

	// Second join rejected
	_, err := svc.Join(testCtx, customer.ID, event.ID)
	if !errors.Is(err, domain.ErrAlreadyInQueue) {
		t.Fatalf("expected ErrAlreadyInQueue, got: %v", err)
	}
}

func TestQueueService_Join_PositionIsAccurate(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildQueueService(t)

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)

	// Three customers join in sequence
	positions := make([]int64, 3)
	for i := range 3 {
		customer := testhelpers.SeedCustomer(testCtx, t, testPool)
		result, err := svc.Join(testCtx, customer.ID, event.ID)
		if err != nil {
			t.Fatalf("customer %d join failed: %v", i+1, err)
		}
		positions[i] = result.Position
	}

	// Positions should be 1, 2, 3 in order
	for i, pos := range positions {
		if pos != int64(i+1) {
			t.Fatalf("expected position %d, got %d", i+1, pos)
		}
	}
}

func TestQueueService_Abandon_WaitingEntry(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildQueueService(t)

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	if _, err := svc.Join(testCtx, customer.ID, event.ID); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	if err := svc.Abandon(testCtx, customer.ID, event.ID); err != nil {
		t.Fatalf("expected abandon to succeed, got: %v", err)
	}

	// Position should now be -1 — not in queue
	pos, err := svc.GetPosition(testCtx, customer.ID, event.ID)
	if err != nil && !errors.Is(err, domain.ErrQueueEntryNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}
	if pos != 0 {
		t.Fatalf("expected position 0 after abandon, got %d", pos)
	}
}

func TestQueueService_Abandon_AdmittedEntryFails(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	svc := buildQueueService(t)

	event := testhelpers.SeedActiveEvent(testCtx, t, testPool)
	customer := testhelpers.SeedCustomer(testCtx, t, testPool)

	// Seed an ADMITTED queue entry directly
	now := time.Now()
	entry := &domain.QueueEntry{
		ID:         uuid.NewString(),
		EventID:    event.ID,
		CustomerID: customer.ID,
		Status:     domain.QueueEntryStatusAdmitted,
		JoinedAt:   now,
		AdmittedAt: &now,
		UpdatedAt:  now,
	}
	postgres.NewQueueStore(postgres.NewDB(testPool)).Create(testCtx, entry)

	// Abandon should fail — ADMITTED can't be abandoned
	err := svc.Abandon(testCtx, customer.ID, event.ID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}
