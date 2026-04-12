// FairQueue — fair inventory allocation for high-demand live events.
//
//	@title			FairQueue API
//	@version		1.0
//	@description	Fair inventory allocation for high-demand live events in Nigeria.
//	@host			localhost:8080
//	@BasePath		/
//
//	@securityDefinitions.apikey	OrganizerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT issued on organizer login. Format: Bearer {token}
//
//	@securityDefinitions.apikey	CustomerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT issued on OTP verification. Format: Bearer {token}
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/DanielPopoola/fairqueue/docs"
	"github.com/DanielPopoola/fairqueue/internal/api"
	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/config"
	"github.com/DanielPopoola/fairqueue/internal/gateway/paystack"
	"github.com/DanielPopoola/fairqueue/internal/infra/migrate"
	"github.com/DanielPopoola/fairqueue/internal/infra/retry"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/worker"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// ── Config ────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := cfg.Logger.NewLogger()

	// ── Infrastructure ────────────────────────────────────────
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

	// ── Stores ────────────────────────────────────────────────
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

	// ── Coordinators ──────────────────────────────────────────
	inventory := service.NewInventoryCoordinator(inventoryStore, lockStore, logger)
	queue := service.NewQueueCoordinator(queueStore, redisQueue, logger)

	// ── Auth ──────────────────────────────────────────────────
	admissionTTL := cfg.Auth.TokenTTL
	orgTokenizer := auth.NewOrganizerTokenizer(cfg.Auth.TokenSecret, 24*time.Hour)
	custTokenizer := auth.NewCustomerTokenizer(cfg.Auth.TokenSecret, 24*time.Hour, admissionTTL)
	admissionTokenizer := auth.NewTokenizer(cfg.Auth.TokenSecret, admissionTTL)

	// ── Gateway ───────────────────────────────────────────────
	retryCfg := retry.Config{
		MaxAttempts: cfg.GatewayRetry.MaxAttempts,
		BaseDelay:   cfg.GatewayRetry.BaseDelay,
		MaxDelay:    cfg.GatewayRetry.MaxDelay,
	}
	gateway := paystack.NewClient(cfg.Paystack.SecretKey, cfg.Paystack.BaseURL, retryCfg)

	// ── Services ──────────────────────────────────────────────
	eventSvc := service.NewEventService(eventStore, logger)
	claimSvc := service.NewClaimService(claimStore, eventStore, inventory, queue, admissionTokenizer, logger)
	queueSvc := service.NewQueueService(eventStore, customerStore, queue, logger)
	paymentSvc := service.NewPaymentService(paymentStore, claimStore, customerStore, eventStore, db, gateway, inventory, logger)

	// ── WebSocket hub ─────────────────────────────────────────
	hub := api.NewHub()

	// ── Workers ───────────────────────────────────────────────
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

	// ── Redis restart recovery ────────────────────────────────
	// Runs once before worker loops begin. Safe to call even if Redis
	// was not wiped — SET NX on inventory and ErrAlreadyInQueue on
	// queue re-adds are both no-ops when keys already exist.
	logger.Info("running Redis state recovery")
	if err := worker.RecoverRedisState(ctx, eventStore, claimStore, queueStore, inventory, redisQueue, logger); err != nil {
		// Non-fatal — reconciliation worker heals any remaining divergence
		logger.Warn("Redis state recovery encountered errors", "error", err)
	}

	// ── API ───────────────────────────────────────────────────
	otpStore := api.NewOTPStore(redisClient)

	handlers := api.NewHandlers(
		eventSvc,
		claimSvc,
		queueSvc,
		paymentSvc,
		customerStore,
		organizerStore,
		claimStore,
		orgTokenizer,
		custTokenizer,
		otpStore,
		hub,
		logger,
	).WithHealthDeps(pgPool, redisstore.NewPinger(rdb))

	router := api.NewRouter(handlers, orgTokenizer, custTokenizer)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// ── Graceful shutdown ─────────────────────────────────────
	// errgroup cancels all workers if any one exits with an error.
	// The HTTP server shuts down cleanly on SIGINT / SIGTERM.
	g, gCtx := errgroup.WithContext(ctx)

	// Workers
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

	// HTTP server
	g.Go(func() error {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	// Shutdown trigger — SIGINT or SIGTERM, or errgroup context cancelled
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

		logger.Info("shutting down HTTP server")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("server exited with error: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}
