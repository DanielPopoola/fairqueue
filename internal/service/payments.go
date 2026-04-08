package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/gateway"
	"github.com/DanielPopoola/fairqueue/internal/gateway/paystack"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
)

type PaymentService struct {
	payments  *postgres.PaymentStore
	claims    *postgres.ClaimStore
	customers *postgres.CustomerStore
	events    *postgres.EventStore
	db        *postgres.DB
	gateway   gateway.PaymentGateway
	inventory *InventoryCoordinator
	logger    *slog.Logger
}

type InitializeResult struct {
	Payment          *domain.Payment
	AuthorizationURL string
}

func (s *PaymentService) Initialize(ctx context.Context, claimID, customerID string) (*InitializeResult, error) {
	claim, customer, event, err := s.fetchInitData(ctx, claimID, customerID)
	if err != nil {
		return nil, err
	}

	if err := s.validateClaimForPayment(claim, customerID); err != nil {
		return nil, err
	}

	ref := "fq-" + uuid.NewString()
	payment := &domain.Payment{
		ID:         uuid.NewString(),
		ClaimID:    claimID,
		CustomerID: customerID,
		Amount:     event.Price,
		Status:     domain.PaymentStatusInitializing,
		Reference:  &ref,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.payments.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("creating payment record: %w", err)
	}

	log := s.logger.With("payment_id", payment.ID, "reference", *payment.Reference)

	resp, err := s.gateway.InitializeTransaction(ctx, s.toGatewayRequest(customer.Email, payment))
	if err != nil {
		return nil, s.handleGatewayInitError(ctx, payment, err, log)
	}

	if err := s.payments.MarkPending(ctx, payment.ID, resp.AuthorizationURL); err != nil {
		log.Warn("failed to mark pending; reconciliation will heal", "error", err)
	}

	return &InitializeResult{Payment: payment, AuthorizationURL: resp.AuthorizationURL}, nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	if !s.gateway.VerifyWebhookSignature(payload, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	event, data, err := s.parseWebhook(payload)
	if err != nil {
		return err
	}

	switch event {
	case "charge.success":
		return s.processChargeSuccess(ctx, data)
	case "charge.failed":
		return s.processChargeFailure(ctx, data)
	default:
		s.logger.Info("ignoring unknown webhook event", "event", event)
		return nil
	}
}

func (s *PaymentService) Reconcile(ctx context.Context, olderThan time.Duration) error {
	stale, err := s.payments.GetStalePayments(ctx, olderThan)
	if err != nil {
		return fmt.Errorf("fetching stale payments: %w", err)
	}

	for i := range stale {
		p := &stale[i]
		if err := s.reconcileSingle(ctx, p); err != nil {
			s.logger.Error("reconciliation failed", "payment_id", p.ID, "error", err)
		}
	}
	return nil
}

func (s *PaymentService) reconcileSingle(ctx context.Context, p *domain.Payment) error {
	switch p.Status { //nolint:exhaustive // Only need to reconcile intermediate states.
	case domain.PaymentStatusInitializing:
		return s.retryInitialization(ctx, p)
	case domain.PaymentStatusPending:
		return s.pollGatewayStatus(ctx, p)
	default:
		return nil
	}
}

// ─── INTERNAL HELPERS (TRANSACTIONS & FLOWS) ─────────────────────────────────

func (s *PaymentService) confirmFlow(ctx context.Context, p *domain.Payment) error {
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.payments.WithTx(tx).UpdateStatus(ctx, p.ID, domain.PaymentStatusConfirmed, domain.PaymentStatusPending); err != nil {
			if errors.Is(err, domain.ErrPaymentNotFound) {
				return nil
			} // Idempotent
			return err
		}
		return s.claims.WithTx(tx).UpdateStatus(ctx, p.ClaimID, domain.ClaimStatusConfirmed, domain.ClaimStatusClaimed)
	})
}

func (s *PaymentService) failureFlow(ctx context.Context, p *domain.Payment, reason string) error {
	err := s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.payments.WithTx(tx).MarkFailed(ctx, p.ID, reason); err != nil {
			if errors.Is(err, domain.ErrPaymentNotFound) {
				return nil
			}
			return err
		}
		return s.claims.WithTx(tx).UpdateStatus(ctx, p.ClaimID, domain.ClaimStatusReleased, domain.ClaimStatusClaimed)
	})

	if err == nil {
		s.restoreInventoryBestEffort(ctx, p.ClaimID)
	}
	return err
}

func (s *PaymentService) fetchInitData(ctx context.Context, claimID, custID string) (*domain.Claim, *domain.Customer, *domain.Event, error) {
	claim, err := s.claims.GetByID(ctx, claimID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading claim: %w", err)
	}

	customer, err := s.customers.GetByID(ctx, custID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading customer: %w", err)
	}

	event, err := s.events.GetByID(ctx, claim.EventID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading event: %w", err)
	}

	return claim, customer, event, nil
}

func (s *PaymentService) validateClaimForPayment(c *domain.Claim, customerID string) error {
	if c.CustomerID != customerID {
		return domain.ErrClaimNotFound
	}
	if c.IsExpired() {
		return domain.ErrClaimExpired
	}
	if c.Status != domain.ClaimStatusClaimed {
		return domain.ErrClaimNotClaimable
	}
	return nil
}

func (s *PaymentService) handleGatewayInitError(ctx context.Context, p *domain.Payment, err error, log *slog.Logger) error {
	if errors.Is(err, paystack.ErrPermanent) {
		if ferr := s.failureFlow(ctx, p, err.Error()); ferr != nil {
			log.Error("failed to handle permanent failure cleanup", "error", ferr)
		}
		return fmt.Errorf("payment permanently rejected: %w", err)
	}
	log.Warn("transient gateway error; reconciliation will retry", "error", err)
	return fmt.Errorf("initialization failed transiently: %w", err)
}

func (s *PaymentService) pollGatewayStatus(ctx context.Context, p *domain.Payment) error {
	resp, err := s.gateway.VerifyTransaction(ctx, *p.Reference)
	if err != nil {
		return err
	}

	switch resp.Status {
	case "success":
		return s.confirmFlow(ctx, p)
	case "failed", "abandoned":
		return s.failureFlow(ctx, p, resp.GatewayResponse)
	default:
		return nil
	}
}

func (s *PaymentService) restoreInventoryBestEffort(ctx context.Context, claimID string) {
	claim, err := s.claims.GetByID(ctx, claimID)
	if err == nil {
		if err := s.inventory.Increment(ctx, claim.EventID); err != nil {
			s.logger.Warn("failed to restore inventory", "claim_id", claimID, "error", err)
		}
	}
}

func (s *PaymentService) parseWebhook(payload []byte) (string, gateway.WebhookData, error) {
	var raw struct {
		Event string `json:"event"`
		Data  struct {
			Reference       string `json:"reference"`
			Status          string `json:"status"`
			GatewayResponse string `json:"gateway_response"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return "", gateway.WebhookData{}, fmt.Errorf("parsing webhook: %w", err)
	}
	return raw.Event, gateway.WebhookData{
		Reference:       raw.Data.Reference,
		Status:          raw.Data.Status,
		GatewayResponse: raw.Data.GatewayResponse,
	}, nil
}

func (s *PaymentService) processChargeSuccess(ctx context.Context, d gateway.WebhookData) error {
	p, err := s.payments.GetByReference(ctx, d.Reference)
	if err != nil {
		return err
	}
	return s.confirmFlow(ctx, p)
}

func (s *PaymentService) processChargeFailure(ctx context.Context, d gateway.WebhookData) error {
	p, err := s.payments.GetByReference(ctx, d.Reference)
	if err != nil {
		return err
	}
	return s.failureFlow(ctx, p, d.GatewayResponse)
}

func (s *PaymentService) toGatewayRequest(email string, p *domain.Payment) gateway.InitializeRequest {
	return gateway.InitializeRequest{
		Email:      email,
		AmountKobo: p.Amount,
		Reference:  *p.Reference,
	}
}

func (s *PaymentService) retryInitialization(ctx context.Context, p *domain.Payment) error {
	_, customer, _, err := s.fetchInitData(ctx, p.ClaimID, p.CustomerID)
	if err != nil {
		return err
	}

	resp, err := s.gateway.InitializeTransaction(ctx, s.toGatewayRequest(customer.Email, p))
	if err != nil {
		if errors.Is(err, paystack.ErrPermanent) {
			return s.failureFlow(ctx, p, err.Error())
		}
		return err
	}
	return s.payments.MarkPending(ctx, p.ID, resp.AuthorizationURL)
}
