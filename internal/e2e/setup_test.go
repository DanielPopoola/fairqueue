package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/api"
	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/config"
	"github.com/DanielPopoola/fairqueue/internal/gateway/mocks"
	"github.com/DanielPopoola/fairqueue/internal/infra/migrate"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
	"github.com/DanielPopoola/fairqueue/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

// testEnv holds everything a test needs to interact with the running server.
type testEnv struct {
	client          *testClient
	pool            *pgxpool.Pool
	rdb             *redis.Client
	admissionWorker *worker.AdmissionWorker
	expiryWorker    *worker.ExpiryWorker
	gateway         *mocks.PaymentGateway
	cancel          context.CancelFunc
}

var (
	sharedEnv *testEnv
	testCtx   = context.Background()
)

func TestMain(m *testing.M) {
	env, cleanup, err := setupTestEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e setup failed: %v\n", err)
		os.Exit(1)
	}
	sharedEnv = env

	code := m.Run()

	cleanup()
	os.Exit(code)
}

type nopTestingT struct{}

func (nopTestingT) Logf(format string, args ...any)   {}
func (nopTestingT) Errorf(format string, args ...any) {}
func (nopTestingT) FailNow()                          {}
func (nopTestingT) Cleanup(f func())                  {} // Do nothing

func setupTestEnv() (*testEnv, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())

	// ── Containers ────────────────────────────────────────────
	pg, err := testhelpers.NewPostgresInstance(ctx)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("postgres: %w", err)
	}

	rc, err := testhelpers.NewRedisInstance(ctx)
	if err != nil {
		cancel()
		pg.Close(ctx)
		return nil, nil, fmt.Errorf("redis: %w", err)
	}

	// ── Migrations ────────────────────────────────────────────
	if err := migrate.Run(ctx, pg.Pool, testhelpers.NewTestLogger()); err != nil {
		cancel()
		pg.Close(ctx)
		rc.Close(ctx)
		return nil, nil, fmt.Errorf("migrations: %w", err)
	}

	// ── Stores ────────────────────────────────────────────────
	db := postgres.NewDB(pg.Pool)
	redisClient, err := redisstore.NewClient(ctx, rc.Client)
	if err != nil {
		cancel()
		pg.Close(ctx)
		rc.Close(ctx)
		return nil, nil, fmt.Errorf("redis client: %w", err)
	}

	eventStore := postgres.NewEventStore(db)
	claimStore := postgres.NewClaimStore(db)
	customerStore := postgres.NewCustomerStore(db)
	organizerStore := postgres.NewOrganizerStore(db)
	queueStore := postgres.NewQueueStore(db)
	paymentStore := postgres.NewPaymentStore(db)

	inventoryStore := redisstore.NewInventoryStore(redisClient)
	lockStore := redisstore.NewLockStore(redisClient, 30*time.Second)
	redisQueue := redisstore.NewQueueStore(redisClient)

	// ── Coordinators ──────────────────────────────────────────
	logger := testhelpers.NewTestLogger()
	inv := service.NewInventoryCoordinator(inventoryStore, lockStore, logger)
	queue := service.NewQueueCoordinator(queueStore, redisQueue, logger)

	// ── Auth ──────────────────────────────────────────────────
	const secret = "test-secret-key-that-is-at-least-32-chars"
	orgTokenizer := auth.NewOrganizerTokenizer(secret, 24*time.Hour)
	custTokenizer := auth.NewCustomerTokenizer(secret, 24*time.Hour, 5*time.Minute)
	admissionTokenizer := auth.NewTokenizer(secret, 5*time.Minute)

	// ── Mock gateway ──────────────────────────────────────────
	gw := mocks.NewPaymentGateway(nopTestingT{})

	// ── Services ─────────────────────────────────────────────
	eventSvc := service.NewEventService(eventStore, logger)
	claimSvc := service.NewClaimService(claimStore, eventStore, inv, queue, admissionTokenizer, logger)
	queueSvc := service.NewQueueService(eventStore, customerStore, queue, logger)
	paymentSvc := service.NewPaymentService(paymentStore, claimStore, customerStore, eventStore, db, gw, inv, logger)

	// ── Hub + OTP store ───────────────────────────────────────
	hub := api.NewHub()
	otpStore := api.NewOTPStore(redisClient)

	// ── Workers (no scheduler — tests call RunOnce directly) ──
	admWorker := worker.NewAdmissionWorker(
		eventStore, queue, inv, admissionTokenizer, hub,
		config.AdmissionWorkerConfig{BatchSize: 50},
		logger,
	)
	expWorker := worker.NewExpiryWorker(
		claimStore, queueStore, inv,
		config.ExpiryWorkerConfig{},
		logger,
	)
	reconcWorker := worker.NewReconciliationWorker(
		eventStore, claimStore, nil, inv,
		config.ReconciliationWorkerConfig{StalePaymentAge: 5 * time.Minute},
		logger,
	)
	_ = reconcWorker // used selectively in tests

	// ── Handlers + router ─────────────────────────────────────
	handlers := api.NewHandlers(
		eventSvc, claimSvc, queueSvc, paymentSvc,
		customerStore, organizerStore, claimStore,
		orgTokenizer, custTokenizer,
		otpStore, hub, logger,
	)
	router := api.NewRouter(handlers, orgTokenizer, custTokenizer)

	// ── Start server on a random port ─────────────────────────
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		pg.Close(ctx)
		rc.Close(ctx)
		return nil, nil, fmt.Errorf("listener: %w", err)
	}

	srv := &http.Server{Handler: router}
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-gCtx.Done()
		return srv.Shutdown(context.Background())
	})

	baseURL := "http://" + listener.Addr().String()

	cleanup := func() {
		cancel()
		g.Wait() //nolint:errcheck
		pg.Close(context.Background())
		rc.Close(context.Background())
	}

	env := &testEnv{
		client:          newTestClient(baseURL),
		pool:            pg.Pool,
		rdb:             rc.Client,
		admissionWorker: admWorker,
		expiryWorker:    expWorker,
		gateway:         gw,
		cancel:          cancel,
	}

	return env, cleanup, nil
}

// ── Test client ───────────────────────────────────────────────

type testClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newTestClient(baseURL string) *testClient {
	return &testClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// WithToken returns a new client that sends the given Bearer token.
func (c *testClient) WithToken(token string) *testClient {
	return &testClient{
		baseURL: c.baseURL,
		token:   token,
		http:    c.http,
	}
}

func (c *testClient) POST(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	return c.do(t, http.MethodPost, path, body)
}

func (c *testClient) GET(t *testing.T, path string) *http.Response {
	t.Helper()
	return c.do(t, http.MethodGet, path, nil)
}

func (c *testClient) PUT(t *testing.T, path string) *http.Response {
	t.Helper()
	return c.do(t, http.MethodPut, path, nil)
}

func (c *testClient) DELETE(t *testing.T, path string) *http.Response {
	t.Helper()
	return c.do(t, http.MethodDelete, path, nil)
}

func (c *testClient) do(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshalling request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, c.baseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// ── Response helpers ──────────────────────────────────────────

func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d\nbody: %s", expected, resp.StatusCode, string(body))
	}
}

func decodeBody[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return v
}

// ── Seed helpers ──────────────────────────────────────────────

// seedOrganizer inserts an organizer and returns their login credentials.
func seedOrganizer(ctx context.Context, t *testing.T, pool *pgxpool.Pool) (email, password string) {
	t.Helper()
	password = "supersecret123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	email = fmt.Sprintf("org-%d@test.com", time.Now().UnixNano())
	_, err = pool.Exec(ctx, `
		INSERT INTO organizers (id, name, email, password_hash, created_at)
		VALUES (gen_random_uuid(), 'Test Organizer', $1, $2, NOW())`,
		email, hash,
	)
	if err != nil {
		t.Fatalf("seeding organizer: %v", err)
	}
	return email, password
}

// webhookPayload builds a Paystack charge event payload.
func webhookPayload(event, reference string) map[string]any {
	return map[string]any{
		"event": event,
		"data": map[string]any{
			"reference":        reference,
			"status":           "success",
			"gateway_response": "Approved",
		},
	}
}

// truncateAll resets all tables between tests.
func truncateAll(ctx context.Context, t *testing.T, pool *pgxpool.Pool, rdb *redis.Client) {
	t.Helper()
	testhelpers.Truncate(ctx, t, pool)
	testhelpers.FlushRedis(ctx, t, rdb)
}
