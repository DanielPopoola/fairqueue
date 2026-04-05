package domain

import (
	"testing"
)

func TestPayment_MarkPending(t *testing.T) {
	t.Run("succeeds from INITIALIZING", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusInitializing}
		if err := p.MarkPending("test-auth-url"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.Status != PaymentStatusPending {
			t.Fatalf("expected PENDING, got %s", p.Status)
		}
		if *p.AuthorizationURL != "test-auth-url" {
			t.Fatalf("expected authorization_url to be set")
		}
	})

	t.Run("fails from PENDING", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusPending}
		if err := p.MarkPending("ref"); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestPayment_Confirm(t *testing.T) {
	t.Run("succeeds from PENDING", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusPending}
		if err := p.Confirm(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.Status != PaymentStatusConfirmed {
			t.Fatalf("expected CONFIRMED, got %s", p.Status)
		}
	})

	t.Run("fails from INITIALIZING", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusInitializing}
		if err := p.Confirm(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("fails from FAILED", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusFailed}
		if err := p.Confirm(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestPayment_Fail(t *testing.T) {
	t.Run("succeeds from PENDING", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusPending}
		if err := p.Fail("insufficient funds"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.Status != PaymentStatusFailed {
			t.Fatalf("expected FAILED, got %s", p.Status)
		}
		if *p.FailureReason != "insufficient funds" {
			t.Fatalf("expected failure reason to be set")
		}
	})

	t.Run("fails from CONFIRMED", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusConfirmed}
		err := p.Fail("invalid card")
		if err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
		if p.Status != PaymentStatusConfirmed {
			t.Fatalf("status should not change")
		}
		if p.FailureReason != nil {
			t.Fatalf("expected failure reason to NOT be set")
		}
	})
}
