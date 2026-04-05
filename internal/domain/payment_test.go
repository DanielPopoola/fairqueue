package domain

import (
	"testing"
	"time"
)

func TestPayment_IsReconcilable(t *testing.T) {
	t.Run("true for INITIALIZING stuck beyond retry threshold", func(t *testing.T) {
		p := &Payment{
			Status:    PaymentStatusInitializing,
			CreatedAt: time.Now().Add(-(InitializingRetryThreshold + time.Second)),
		}
		if !p.IsReconcilable() {
			t.Fatal("expected reconcilable")
		}
	})

	t.Run("false for fresh INITIALIZING", func(t *testing.T) {
		p := &Payment{
			Status:    PaymentStatusInitializing,
			CreatedAt: time.Now(),
		}
		if p.IsReconcilable() {
			t.Fatal("expected not reconcilable")
		}
	})

	t.Run("true for PENDING stuck beyond reconciliation threshold", func(t *testing.T) {
		p := &Payment{
			Status:    PaymentStatusPending,
			UpdatedAt: time.Now().Add(-(PendingReconciliationThreshold + time.Second)),
		}
		if !p.IsReconcilable() {
			t.Fatal("expected reconcilable")
		}
	})

	t.Run("false for fresh PENDING", func(t *testing.T) {
		p := &Payment{
			Status:    PaymentStatusPending,
			UpdatedAt: time.Now(),
		}
		if p.IsReconcilable() {
			t.Fatal("expected not reconcilable")
		}
	})

	t.Run("false for CONFIRMED", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusConfirmed}
		if p.IsReconcilable() {
			t.Fatal("confirmed payments should never be reconcilable")
		}
	})

	t.Run("false for FAILED", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusFailed}
		if p.IsReconcilable() {
			t.Fatal("failed payments should never be reconcilable")
		}
	})
}

func TestPayment_MarkPending(t *testing.T) {
	t.Run("succeeds from INITIALIZING", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusInitializing}
		if err := p.MarkPending("pay_ref_123"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.Status != PaymentStatusPending {
			t.Fatalf("expected PENDING, got %s", p.Status)
		}
		if p.PaystackReference != "pay_ref_123" {
			t.Fatalf("expected reference to be set")
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
		if err := p.Fail(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.Status != PaymentStatusFailed {
			t.Fatalf("expected FAILED, got %s", p.Status)
		}
	})

	t.Run("fails from CONFIRMED", func(t *testing.T) {
		p := &Payment{Status: PaymentStatusConfirmed}
		if err := p.Fail(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}
