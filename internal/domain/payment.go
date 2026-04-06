package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusInitializing PaymentStatus = "INITIALIZING"
	PaymentStatusPending      PaymentStatus = "PENDING"
	PaymentStatusConfirmed    PaymentStatus = "CONFIRMED"
	PaymentStatusFailed       PaymentStatus = "FAILED"
)

type Payment struct {
	ID               string
	ClaimID          string
	CustomerID       string
	Amount           int64
	Status           PaymentStatus
	Reference        *string
	AuthorizationURL *string
	FailureReason    *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// MarkPending transitions from INITIALIZING to PENDING after
// the Paystack initialization call succeeds.
func (p *Payment) MarkPending(authorizationURL string) error {
	if p.Status != PaymentStatusInitializing {
		return ErrInvalidTransition
	}
	p.AuthorizationURL = &authorizationURL
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
func (p *Payment) Fail(reason string) error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidTransition
	}
	p.Status = PaymentStatusFailed
	p.FailureReason = &reason
	p.UpdatedAt = time.Now()
	return nil
}
