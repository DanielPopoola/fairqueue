package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/api"
	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/config"
	mockgateway "github.com/DanielPopoola/fairqueue/internal/gateway/mock"
	"github.com/DanielPopoola/fairqueue/internal/infra/migrate"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := cfg.Logger.NewLogger()
	ctx := context.Background()

	pgPool, err := cfg.Database.Pool(ctx)
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pgPool.Close()

	logger.Info("running migrations")
	if err := migrate.Run(ctx, pgPool, logger); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	rdb := cfg.Redis.Client()
	redisClient, err := redisstore.NewClient(ctx, rdb)
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}

	db := postgres.NewDB(pgPool)
	eventStore := postgres.NewEventStore(db)
	claimStore := postgres.NewClaimStore(db)
	customerStore := postgres.NewCustomerStore(db)
	organizerStore := postgres.NewOrganizerStore(db)
	queueStore := postgres.NewQueueStore(db)
	paymentStore := postgres.NewPaymentStore(db)

	inventoryStore := redisstore.NewInventoryStore(redisClient)
	lockStore := redisstore.NewLockStore(redisClient, 30*time.Second)
	redisQueue := redisstore.NewQueueStore(redisClient)

	inventory := service.NewInventoryCoordinator(inventoryStore, lockStore, logger)
	queue := service.NewQueueCoordinator(queueStore, redisQueue, logger)

	admissionTTL := cfg.Auth.TokenTTL
	orgTokenizer := auth.NewOrganizerTokenizer(cfg.Auth.TokenSecret, 24*time.Hour)
	custTokenizer := auth.NewCustomerTokenizer(cfg.Auth.TokenSecret, 24*time.Hour, admissionTTL)
	admissionTokenizer := auth.NewTokenizer(cfg.Auth.TokenSecret, admissionTTL)

	gw := mockgateway.NewGateway()

	eventSvc := service.NewEventService(eventStore, logger)
	claimSvc := service.NewClaimService(claimStore, eventStore, inventory, queue, admissionTokenizer, logger)
	queueSvc := service.NewQueueService(eventStore, customerStore, queue, logger)
	paymentSvc := service.NewPaymentService(paymentStore, claimStore, customerStore, eventStore, db, gw, inventory, logger)

	hub := api.NewHub()

	reconciliationWorker := worker.NewReconciliationWorker(
		eventStore, claimStore, paymentSvc, inventory,
		cfg.Workers.Reconciliation,
		logger,
	)
	expiryWorker := worker.NewExpiryWorker(
		claimStore, queueStore, inventory,
		cfg.Workers.Expiry,
		logger,
	)
	admissionWorker := worker.NewAdmissionWorker(
		eventStore, queue, inventory, admissionTokenizer, hub,
		cfg.Workers.Admission,
		logger,
	)

	logger.Info("running Redis state recovery")
	if err := worker.RecoverRedisState(ctx, eventStore, claimStore, queueStore, inventory, redisQueue, logger); err != nil {
		logger.Warn("Redis state recovery encountered errors", "error", err)
	}

	otpStore := api.NewOTPStore(redisClient)
	handlers := api.NewHandlers(
		eventSvc, claimSvc, queueSvc, paymentSvc,
		customerStore, organizerStore, claimStore,
		orgTokenizer, custTokenizer,
		otpStore, hub, logger,
	).WithHealthDeps(pgPool, redisstore.NewPinger(rdb))

	router := api.NewRouter(handlers, orgTokenizer, custTokenizer)

	// Wrap the production router with load test helper endpoints
	// These routes never exist in cmd/api/main.go
	mux := buildMux(router, pgPool, rdb)

	loadTestPort := resolveLoadTestPort()
	srv := &http.Server{
		Addr:         ":" + loadTestPort,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		s := worker.NewScheduler("reconciliation", cfg.Workers.Reconciliation.Interval, reconciliationWorker.Run, logger)
		return s.Start(gCtx)
	})
	g.Go(func() error {
		s := worker.NewScheduler("expiry", cfg.Workers.Expiry.Interval, expiryWorker.Run, logger)
		return s.Start(gCtx)
	})
	g.Go(func() error {
		s := worker.NewScheduler("admission", cfg.Workers.Admission.Interval, admissionWorker.Run, logger)
		return s.Start(gCtx)
	})
	g.Go(func() error {
		logger.Info("load test server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(quit)
		select {
		case sig := <-quit:
			logger.Info("shutdown signal received", "signal", sig)
		case <-gCtx.Done():
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("server exited with error: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

// buildMux wraps the production router and adds load-test-only endpoints.
// These endpoints expose internals (OTP values, DB counts, seeding)
// that must never exist in the production server.
func buildMux(router http.Handler, pgPool *pgxpool.Pool, rdb *redis.Client) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", router)

	// POST /loadtest/seed — creates a known organizer for k6 setup()
	mux.HandleFunc("/loadtest/seed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		hash, err := auth.HashPassword("loadtest-password")
		if err != nil {
			http.Error(w, "hashing failed", http.StatusInternalServerError)
			return
		}

		_, err = pgPool.Exec(r.Context(), `
            INSERT INTO organizers (id, name, email, password_hash, created_at)
            VALUES (gen_random_uuid(), 'Load Test Organizer', 'loadtest@admin.com', $1, NOW())
            ON CONFLICT (email) DO NOTHING
        `, hash)
		if err != nil {
			http.Error(w, fmt.Sprintf("seed failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"seeded":true,"email":"loadtest@admin.com","password":"loadtest-password"}`)
	})

	// GET /loadtest/otp?email=x — reads OTP from Redis so k6 can verify without email
	mux.HandleFunc("/loadtest/otp", func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "email required", http.StatusBadRequest)
			return
		}

		otp, err := rdb.Get(r.Context(), "otp:"+email).Result()
		if err != nil {
			http.Error(w, "otp not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"otp":%q}`, otp)
	})

	// GET /loadtest/stats?event_id=x — claim counts for correctness assertion
	mux.HandleFunc("/loadtest/stats", func(w http.ResponseWriter, r *http.Request) {
		eventID := r.URL.Query().Get("event_id")
		if eventID == "" {
			http.Error(w, "event_id required", http.StatusBadRequest)
			return
		}

		var totalInventory int
		var claimed, confirmed, released int64

		pgPool.QueryRow(r.Context(),
			"SELECT total_inventory FROM events WHERE id = $1", eventID,
		).Scan(&totalInventory)

		pgPool.QueryRow(r.Context(),
			"SELECT COUNT(*) FROM claims WHERE event_id = $1 AND status = 'CLAIMED'", eventID,
		).Scan(&claimed)

		pgPool.QueryRow(r.Context(),
			"SELECT COUNT(*) FROM claims WHERE event_id = $1 AND status = 'CONFIRMED'", eventID,
		).Scan(&confirmed)

		pgPool.QueryRow(r.Context(),
			"SELECT COUNT(*) FROM claims WHERE event_id = $1 AND status = 'RELEASED'", eventID,
		).Scan(&released)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "total_inventory":%d,
            "claimed_count":%d,
            "confirmed_count":%d,
            "released_count":%d
        }`, totalInventory, claimed, confirmed, released)
	})

	return mux
}

func resolveLoadTestPort() string {
	if port := strings.TrimSpace(os.Getenv("LOADTEST_SERVER_PORT")); port != "" {
		return port
	}
	return "8082"
}
