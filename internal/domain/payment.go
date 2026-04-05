package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusInitializing PaymentStatus = "INITIALIZING"
	PaymentStatusPending      PaymentStatus = "PENDING"
	PaymentStatusConfirmed    PaymentStatus = "CONFIRMED"
	PaymentStatusFailed       PaymentStatus = "FAILED"
)

// PendingReconciliationThreshold is how long a PENDING payment
// can sit without a webhook before the reconciliation worker
// polls Paystack to check its real state.
const PendingReconciliationThreshold = 3 * time.Minute

// InitializingRetryThreshold is how long an INITIALIZING payment
// can sit before the reconciliation worker retries the Paystack call.
const InitializingRetryThreshold = 2 * time.Minute

type Payment struct {
	ID                string
	ClaimID           string
	EventID           string
	UserID            string
	AmountKobo        int64
	Status            PaymentStatus
	PaystackReference string // set after Paystack call succeeds
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsReconcilable returns true if this payment is stuck in a
// state that the reconciliation worker should act on.
func (p *Payment) IsReconcilable() bool {
	switch p.Status { //nolint:exhaustive // only payments stuck in initializing or pending need reconciliation
	case PaymentStatusInitializing:
		return time.Since(p.CreatedAt) > InitializingRetryThreshold
	case PaymentStatusPending:
		return time.Since(p.UpdatedAt) > PendingReconciliationThreshold
	default:
		return false
	}
}

// MarkPending transitions from INITIALIZING to PENDING after
// the Paystack initialization call succeeds.
func (p *Payment) MarkPending(paystackReference string) error {
	if p.Status != PaymentStatusInitializing {
		return ErrInvalidTransition
	}
	p.PaystackReference = paystackReference
	p.Status = PaymentStatusPending
	p.UpdatedAt = time.Now()
	return nil
}

// Confirm transitions from PENDING to CONFIRMED.
// Called by webhook handler or reconciliation worker.
func (p *Payment) Confirm() error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidTransition
	}
	p.Status = PaymentStatusConfirmed
	p.UpdatedAt = time.Now()
	return nil
}

// Fail transitions from PENDING to FAILED.
// Called by webhook handler or reconciliation worker.
func (p *Payment) Fail() error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidTransition
	}
	p.Status = PaymentStatusFailed
	p.UpdatedAt = time.Now()
	return nil
}
