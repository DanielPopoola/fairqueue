package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/gateway"
	"github.com/DanielPopoola/fairqueue/internal/gateway/mocks"
	"github.com/DanielPopoola/fairqueue/internal/gateway/paystack"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
)

// buildPaymentService wires the PaymentService with a real DB/Redis
// but a programmable mock gateway.
func buildPaymentService(t *testing.T, gw gateway.PaymentGateway) *service.PaymentService {
	t.Helper()

	db := postgres.NewDB(testPool)
	redisClient, err := redisstore.NewClient(testCtx, testClient)
	if err != nil {
		t.Fatalf("creating redis client: %v", err)
	}

	logger := testhelpers.NewTestLogger()
	inventoryStore := redisstore.NewInventoryStore(redisClient)
	lockStore := redisstore.NewLockStore(redisClient, 30*time.Second)
	inventory := service.NewInventoryCoordinator(inventoryStore, lockStore, logger)

	return service.NewPaymentService(
		postgres.NewPaymentStore(db),
		postgres.NewClaimStore(db),
		postgres.NewCustomerStore(db),
		postgres.NewEventStore(db),
		db,
		gw,
		inventory,
		logger,
	)
}

// seedClaimWithPaymentReady seeds an active event, customer, and a CLAIMED
// claim — the minimum state required before Initialize can be called.
func seedClaimWithPaymentReady(ctx context.Context, t *testing.T, pool *pgxpool.Pool) (*domain.Claim, *domain.Customer) {
	t.Helper()
	event := testhelpers.SeedActiveEvent(ctx, t, pool)
	customer := testhelpers.SeedCustomer(ctx, t, pool)
	claim := testhelpers.SeedClaim(ctx, t, pool, customer.ID, event.ID)
	return claim, customer
}

// ─────────────────────────────────────────────────────────────
// Group 1: Initialize — persist-before-gateway logic
// ─────────────────────────────────────────────────────────────

func TestPaymentService_Initialize_HappyPath(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("InitializeTransaction", testCtx, mock.AnythingOfType("gateway.InitializeRequest")).
		Return(&gateway.InitializeResponse{
			AuthorizationURL: "https://paystack.co/pay/test123",
			Reference:        "fq-test-ref",
		}, nil)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	result, err := svc.Initialize(testCtx, claim.ID, customer.ID)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.AuthorizationURL != "https://paystack.co/pay/test123" {
		t.Fatalf("unexpected authorization_url: %s", result.AuthorizationURL)
	}

	// DB record must be PENDING — not still INITIALIZING
	db := postgres.NewDB(testPool)
	stored, err := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
	if err != nil {
		t.Fatalf("fetching payment: %v", err)
	}
	if stored.Status != domain.PaymentStatusPending {
		t.Fatalf("expected PENDING, got %s", stored.Status)
	}
	if stored.AuthorizationURL == nil || *stored.AuthorizationURL != "https://paystack.co/pay/test123" {
		t.Fatal("expected authorization_url to be persisted")
	}
}

func TestPaymentService_Initialize_TransientGatewayError_RecordStaysInitializing(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("InitializeTransaction", testCtx, mock.AnythingOfType("gateway.InitializeRequest")).
		Return(nil, fmt.Errorf("%w: upstream timeout", paystack.ErrTransient))

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	_, err := svc.Initialize(testCtx, claim.ID, customer.ID)
	if err == nil {
		t.Fatal("expected error on transient failure")
	}

	// Record must stay INITIALIZING so the reconciliation worker can find and retry it
	db := postgres.NewDB(testPool)
	stored, err := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
	if err != nil {
		t.Fatalf("expected DB record to exist, got: %v", err)
	}
	if stored.Status != domain.PaymentStatusInitializing {
		t.Fatalf("expected INITIALIZING, got %s — reconciliation worker would miss this", stored.Status)
	}
}

func TestPaymentService_Initialize_PermanentGatewayError_ClaimReleased(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("InitializeTransaction", testCtx, mock.AnythingOfType("gateway.InitializeRequest")).
		Return(nil, fmt.Errorf("%w: invalid email address", paystack.ErrPermanent))

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	// Prime Redis so inventory rollback has a counter to increment
	rc, _ := redisstore.NewClient(testCtx, testClient)
	redisstore.NewInventoryStore(rc).Set(testCtx, claim.EventID, 10)

	_, err := svc.Initialize(testCtx, claim.ID, customer.ID)
	if err == nil {
		t.Fatal("expected error on permanent failure")
	}

	db := postgres.NewDB(testPool)

	payment, err := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
	if err != nil {
		t.Fatalf("fetching payment: %v", err)
	}
	if payment.Status != domain.PaymentStatusFailed {
		t.Fatalf("expected payment FAILED, got %s", payment.Status)
	}

	// Claim must be released so the inventory returns to the pool
	released, err := postgres.NewClaimStore(db).GetByID(testCtx, claim.ID)
	if err != nil {
		t.Fatalf("fetching claim: %v", err)
	}
	if released.Status != domain.ClaimStatusReleased {
		t.Fatalf("expected claim RELEASED, got %s", released.Status)
	}
}

func TestPaymentService_Initialize_Idempotent_ReturnsExistingPayment(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	// .Once() — the mock will panic if called a second time
	gw := mocks.NewPaymentGateway(t)
	gw.On("InitializeTransaction", testCtx, mock.AnythingOfType("gateway.InitializeRequest")).
		Return(&gateway.InitializeResponse{
			AuthorizationURL: "https://paystack.co/pay/original",
			Reference:        "fq-original",
		}, nil).Once()

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	first, err := svc.Initialize(testCtx, claim.ID, customer.ID)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call must short-circuit before hitting the gateway
	second, err := svc.Initialize(testCtx, claim.ID, customer.ID)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if first.Payment.ID != second.Payment.ID {
		t.Fatal("expected second Initialize to return the same payment record")
	}
}

// ─────────────────────────────────────────────────────────────
// Group 2: Webhook handling
// ─────────────────────────────────────────────────────────────

func TestPaymentService_HandleWebhook_Success_ConfirmsBoth(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("VerifyWebhookSignature", mock.Anything, "valid-sig").Return(true)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	ref := "fq-webhook-ref"
	payment := seedPendingPayment(testCtx, t, testPool, claim.ID, customer.ID, ref)

	if err := svc.HandleWebhook(testCtx, webhookPayload("charge.success", ref), "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	db := postgres.NewDB(testPool)
	awaitWebhookEffects(t, func() bool {
		p, _ := postgres.NewPaymentStore(db).GetByID(testCtx, payment.ID)
		c, _ := postgres.NewClaimStore(db).GetByID(testCtx, claim.ID)
		return p.Status == domain.PaymentStatusConfirmed && c.Status == domain.ClaimStatusConfirmed
	})
}

func TestPaymentService_HandleWebhook_InvalidSignature_NoStateChange(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("VerifyWebhookSignature", mock.Anything, "garbage-sig").Return(false)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)
	ref := "fq-sig-ref"
	seedPendingPayment(testCtx, t, testPool, claim.ID, customer.ID, ref)

	err := svc.HandleWebhook(testCtx, webhookPayload("charge.success", ref), "garbage-sig")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}

	// Payment must still be PENDING — invalid sig must not cause any state change
	db := postgres.NewDB(testPool)
	stored, _ := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
	if stored.Status != domain.PaymentStatusPending {
		t.Fatalf("expected PENDING after invalid sig, got %s", stored.Status)
	}
}

func TestPaymentService_HandleWebhook_Failure_ReleasesClaimAndRestoresInventory(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("VerifyWebhookSignature", mock.Anything, "valid-sig").Return(true)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	rc, _ := redisstore.NewClient(testCtx, testClient)
	redisstore.NewInventoryStore(rc).Set(testCtx, claim.EventID, 5)

	ref := "fq-fail-ref"
	payment := seedPendingPayment(testCtx, t, testPool, claim.ID, customer.ID, ref)

	// charge.failed payload — gateway_response carries the reason
	payload := fmt.Appendf(nil,
		`{"event":"charge.failed","data":{"reference":%q,"status":"failed","gateway_response":"Insufficient funds"}}`,
		ref,
	)

	if err := svc.HandleWebhook(testCtx, payload, "valid-sig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	db := postgres.NewDB(testPool)
	awaitWebhookEffects(t, func() bool {
		p, _ := postgres.NewPaymentStore(db).GetByID(testCtx, payment.ID)
		c, _ := postgres.NewClaimStore(db).GetByID(testCtx, claim.ID)
		count, _ := redisstore.NewInventoryStore(rc).Get(testCtx, claim.EventID)
		return p.Status == domain.PaymentStatusFailed && c.Status == domain.ClaimStatusReleased && count == 6
	})
}

func TestPaymentService_HandleWebhook_DuplicateSuccess_Idempotent(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("VerifyWebhookSignature", mock.Anything, "valid-sig").Return(true)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)
	ref := "fq-dup-ref"
	seedPendingPayment(testCtx, t, testPool, claim.ID, customer.ID, ref)

	payload := webhookPayload("charge.success", ref)

	if err := svc.HandleWebhook(testCtx, payload, "valid-sig"); err != nil {
		t.Fatalf("first webhook: %v", err)
	}

	// Second identical webhook — must not error, must not double-process
	if err := svc.HandleWebhook(testCtx, payload, "valid-sig"); err != nil {
		t.Fatalf("duplicate webhook returned error: %v", err)
	}

	db := postgres.NewDB(testPool)
	awaitWebhookEffects(t, func() bool {
		p, _ := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
		return p.Status == domain.PaymentStatusConfirmed
	})
}

// ─────────────────────────────────────────────────────────────
// Group 3: Concurrency
// ─────────────────────────────────────────────────────────────

func TestPaymentService_DoubleConfirmRace_BothReturnNil(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("VerifyWebhookSignature", mock.Anything, "valid-sig").Return(true)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)
	ref := "fq-race-ref"
	seedPendingPayment(testCtx, t, testPool, claim.ID, customer.ID, ref)

	payload := webhookPayload("charge.success", ref)

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for range 2 {
		wg.Go(func() {
			err := svc.HandleWebhook(testCtx, payload, "valid-sig")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
			}
		})
	}

	wg.Wait()

	// Both goroutines must return nil — the WHERE clause absorbs the race gracefully
	if len(errs) > 0 {
		t.Fatalf("expected both goroutines to return nil, got errors: %v", errs)
	}

	db := postgres.NewDB(testPool)
	awaitWebhookEffects(t, func() bool {
		p, _ := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
		return p.Status == domain.PaymentStatusConfirmed
	})
}

func TestPaymentService_Initialize_UserDoubleClick_ReturnsSameRecord(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	// .Once() — mock panics if called more than once
	gw := mocks.NewPaymentGateway(t)
	gw.On("InitializeTransaction", testCtx, mock.AnythingOfType("gateway.InitializeRequest")).
		Return(&gateway.InitializeResponse{
			AuthorizationURL: "https://paystack.co/pay/once",
			Reference:        "fq-once",
		}, nil).Once()

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*service.InitializeResult
		errs    []error
	)

	for range 5 {
		wg.Go(func() {
			r, err := svc.Initialize(testCtx, claim.ID, customer.ID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
			} else {
				results = append(results, r)
			}
		})
	}

	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// All goroutines must return the same payment ID
	firstID := results[0].Payment.ID
	for _, r := range results[1:] {
		if r.Payment.ID != firstID {
			t.Fatal("different payment IDs returned — double-initialization occurred")
		}
	}
}

// ─────────────────────────────────────────────────────────────
// Group 4: Reconciliation
// ─────────────────────────────────────────────────────────────

func TestPaymentService_Reconcile_HealsInitializing(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	gw := mocks.NewPaymentGateway(t)
	gw.On("InitializeTransaction", testCtx, mock.AnythingOfType("gateway.InitializeRequest")).
		Return(&gateway.InitializeResponse{
			AuthorizationURL: "https://paystack.co/pay/healed",
			Reference:        "fq-healed",
		}, nil)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)

	// Simulate a crash that left the record in INITIALIZING
	ref := "fq-stale-init"
	seedStalePayment(testCtx, t, testPool, claim.ID, customer.ID, ref, domain.PaymentStatusInitializing, 10*time.Minute)

	if err := svc.Reconcile(testCtx, 5*time.Minute); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	db := postgres.NewDB(testPool)
	p, _ := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
	if p.Status != domain.PaymentStatusPending {
		t.Fatalf("expected PENDING after healing INITIALIZING, got %s", p.Status)
	}
}

func TestPaymentService_Reconcile_HealsPending_GatewayConfirms(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	ref := "fq-pending-ref"

	gw := mocks.NewPaymentGateway(t)
	gw.On("VerifyTransaction", testCtx, ref).
		Return(&gateway.VerifyResponse{
			Status:          "success",
			GatewayResponse: "Approved",
			Reference:       ref,
		}, nil)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)
	seedStalePayment(testCtx, t, testPool, claim.ID, customer.ID, ref, domain.PaymentStatusPending, 10*time.Minute)

	if err := svc.Reconcile(testCtx, 5*time.Minute); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	db := postgres.NewDB(testPool)
	p, _ := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
	if p.Status != domain.PaymentStatusConfirmed {
		t.Fatalf("expected CONFIRMED, got %s", p.Status)
	}
}

func TestPaymentService_Reconcile_HealsPending_StillInProgress(t *testing.T) {
	testhelpers.Truncate(testCtx, t, testPool)
	testhelpers.FlushRedis(testCtx, t, testClient)

	ref := "fq-inprogress-ref"

	gw := mocks.NewPaymentGateway(t)
	gw.On("VerifyTransaction", testCtx, ref).
		Return(&gateway.VerifyResponse{
			Status:          "ongoing", // user hasn't entered PIN yet
			GatewayResponse: "Pending",
			Reference:       ref,
		}, nil)

	svc := buildPaymentService(t, gw)
	claim, customer := seedClaimWithPaymentReady(testCtx, t, testPool)
	seedStalePayment(testCtx, t, testPool, claim.ID, customer.ID, ref, domain.PaymentStatusPending, 10*time.Minute)

	if err := svc.Reconcile(testCtx, 5*time.Minute); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// Worker must do nothing — record stays PENDING
	db := postgres.NewDB(testPool)
	p, _ := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claim.ID)
	if p.Status != domain.PaymentStatusPending {
		t.Fatalf("expected PENDING (still in progress), got %s", p.Status)
	}
}

// ─────────────────────────────────────────────────────────────
// Seed helpers
// ─────────────────────────────────────────────────────────────

// seedPendingPayment inserts a PENDING payment with a known reference.
// Used by webhook and concurrency tests that need a pre-existing transaction.
func seedPendingPayment(ctx context.Context, t *testing.T, pool *pgxpool.Pool, claimID, customerID, ref string) *domain.Payment {
	t.Helper()
	return seedPaymentWithStatus(ctx, t, pool, claimID, customerID, ref, domain.PaymentStatusPending, 0)
}

// seedStalePayment inserts a payment backdated by age.
// Used by reconciliation tests to simulate records that need healing.
func seedStalePayment(ctx context.Context, t *testing.T, pool *pgxpool.Pool, claimID, customerID, ref string, status domain.PaymentStatus, age time.Duration) *domain.Payment {
	t.Helper()
	return seedPaymentWithStatus(ctx, t, pool, claimID, customerID, ref, status, age)
}

func seedPaymentWithStatus(ctx context.Context, t *testing.T, pool *pgxpool.Pool, claimID, customerID, ref string, status domain.PaymentStatus, age time.Duration) *domain.Payment {
	t.Helper()

	db := postgres.NewDB(pool)
	claim, err := postgres.NewClaimStore(db).GetByID(ctx, claimID)
	if err != nil {
		t.Fatalf("loading claim for payment seed: %v", err)
	}
	event, err := postgres.NewEventStore(db).GetByID(ctx, claim.EventID)
	if err != nil {
		t.Fatalf("loading event for payment seed: %v", err)
	}

	ts := time.Now().Add(-age)
	p := &domain.Payment{
		ID:         uuid.NewString(),
		ClaimID:    claimID,
		CustomerID: customerID,
		Amount:     event.Price,
		Status:     status,
		Reference:  &ref,
		CreatedAt:  ts,
		UpdatedAt:  ts,
	}

	// Insert directly so we control updated_at for stale detection
	_, err = pool.Exec(ctx, `
		INSERT INTO payments (id, claim_id, customer_id, amount_kobo, status, reference, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, p.ClaimID, p.CustomerID, p.Amount, p.Status, p.Reference, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("seeding payment: %v", err)
	}
	return p
}

// webhookPayload produces a minimal Paystack-shaped charge.success body.
func webhookPayload(event, reference string) []byte {
	return []byte(fmt.Sprintf(
		`{"event":%q,"data":{"reference":%q,"status":"success","gateway_response":"Approved"}}`,
		event, reference,
	))
}

func awaitWebhookEffects(t *testing.T, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for asynchronous webhook effects")
}
