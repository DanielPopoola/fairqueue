package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/jackc/pgx/v5"
)

type PaymentStore struct {
	exec Executor
}

func NewPaymentStore(db *DB) *PaymentStore {
	return &PaymentStore{exec: db.Pool}
}

// WithTx returns a new PaymentStore that uses the given transaction.
func (s *PaymentStore) WithTx(tx pgx.Tx) *PaymentStore {
	return &PaymentStore{exec: tx}
}

func (s *PaymentStore) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
        INSERT INTO payments (
            id, claim_id, customer_id, amount_kobo,
            status, reference, failure_reason,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`

	_, err := s.exec.Exec(ctx, query,
		payment.ID,
		payment.ClaimID,
		payment.CustomerID,
		payment.Amount,
		payment.Status,
		payment.Reference,
		payment.FailureReason,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	if err != nil {
		if IsUniqueViolation(err) {
			return domain.ErrPaymentAlreadyMade
		}
		return fmt.Errorf("inserting payment: %w", err)
	}
	return nil
}

func (s *PaymentStore) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	return s.scanOne(ctx, `
        SELECT id, claim_id, customer_id, amount_kobo,
               status, reference, failure_reason,
               authorization_url, created_at, updated_at
        FROM payments
        WHERE id = $1`, id)
}

func (s *PaymentStore) GetByClaimID(ctx context.Context, claimID string) (*domain.Payment, error) {
	return s.scanOne(ctx, `
        SELECT id, claim_id, customer_id, amount_kobo,
               status, reference, failure_reason,
               authorization_url, created_at, updated_at
        FROM payments
        WHERE claim_id = $1`, claimID)
}

func (s *PaymentStore) GetByReference(ctx context.Context, reference string) (*domain.Payment, error) {
	return s.scanOne(ctx, `
        SELECT id, claim_id, customer_id, amount_kobo,
               status, reference, failure_reason,
               authorization_url, created_at, updated_at
        FROM payments
        WHERE reference = $1`, reference)
}

func (s *PaymentStore) GetStalePayments(ctx context.Context, olderThan time.Duration) ([]domain.Payment, error) {
	query := `
        SELECT id, claim_id, customer_id, amount_kobo,
               status, reference, failure_reason,
               authorization_url, created_at, updated_at
        FROM payments
        WHERE status IN ('INITIALIZING', 'PENDING')
        AND updated_at < $1
	`
	rows, err := s.exec.Query(ctx, query, time.Now().Add(-olderThan))
	if err != nil {
		return nil, fmt.Errorf("querying stale payments: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Payment])
}

func (s *PaymentStore) UpdateStatus(
	ctx context.Context,
	id string,
	newStatus domain.PaymentStatus,
	expectedStatus domain.PaymentStatus,
) error {
	query := `
        UPDATE payments
        SET status = $1, updated_at = $2
        WHERE id = $3
        AND status = $4
	`
	result, err := s.exec.Exec(ctx, query,
		newStatus,
		time.Now(),
		id,
		expectedStatus,
	)
	if err != nil {
		return fmt.Errorf("updating payment status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrPaymentNotFound
	}
	return nil
}

func (s *PaymentStore) MarkPending(ctx context.Context, id, authorizationURL string) error {
	query := `
        UPDATE payments
        SET status = 'PENDING',
            authorization_url = $1,
            updated_at = $2
        WHERE id = $3
        AND status = 'INITIALIZING'`

	result, err := s.exec.Exec(ctx, query, authorizationURL, time.Now(), id)
	if err != nil {
		return fmt.Errorf("marking payment pending: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrPaymentNotFound
	}
	return nil
}

func (s *PaymentStore) MarkFailed(ctx context.Context, id, reason string, expectedStatus domain.PaymentStatus) error {
	query := `
        UPDATE payments
        SET status = 'FAILED',
            failure_reason = $1,
            updated_at = $2
        WHERE id = $3
        AND status = $4`

	result, err := s.exec.Exec(ctx, query, reason, time.Now(), id, expectedStatus)
	if err != nil {
		return fmt.Errorf("marking payment failed: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrPaymentNotFound
	}
	return nil
}

// scanOne is a private helper that reduces repetition across
// the three single-row payment queries.
func (s *PaymentStore) scanOne(ctx context.Context, query string, arg any) (*domain.Payment, error) {
	var p domain.Payment
	err := s.exec.QueryRow(ctx, query, arg).Scan(
		&p.ID,
		&p.ClaimID,
		&p.CustomerID,
		&p.Amount,
		&p.Status,
		&p.Reference,
		&p.FailureReason,
		&p.AuthorizationURL,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("scanning payment: %w", err)
	}
	return &p, nil
}
