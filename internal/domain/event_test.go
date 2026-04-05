package domain

import (
	"testing"
)

func TestEvent_Activate(t *testing.T) {
	t.Run("succeeds from DRAFT", func(t *testing.T) {
		e := &Event{Status: EventStatusDraft}
		if err := e.Activate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if e.Status != EventStatusActive {
			t.Fatalf("expected ACTIVE, got %s", e.Status)
		}
	})

	t.Run("fails from ACTIVE", func(t *testing.T) {
		e := &Event{Status: EventStatusActive}
		if err := e.Activate(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("fails from SOLD_OUT", func(t *testing.T) {
		e := &Event{Status: EventStatusSoldOut}
		if err := e.Activate(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("fails from ENDED", func(t *testing.T) {
		e := &Event{Status: EventStatusEnded}
		if err := e.Activate(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestEvent_OnInventoryDepleted(t *testing.T) {
	t.Run("transitions to SOLD_OUT when inventory is zero", func(t *testing.T) {
		e := &Event{Status: EventStatusActive, AvailableInventory: 0}
		if err := e.OnInventoryDepleted(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if e.Status != EventStatusSoldOut {
			t.Fatalf("expected SOLD_OUT, got %s", e.Status)
		}
	})

	t.Run("no-op when inventory is above zero", func(t *testing.T) {
		e := &Event{Status: EventStatusActive, AvailableInventory: 1}
		if err := e.OnInventoryDepleted(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if e.Status != EventStatusActive {
			t.Fatalf("expected ACTIVE to be unchanged, got %s", e.Status)
		}
	})

	t.Run("fails when event is not ACTIVE", func(t *testing.T) {
		e := &Event{Status: EventStatusDraft, AvailableInventory: 0}
		if err := e.OnInventoryDepleted(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestEvent_End(t *testing.T) {
	t.Run("succeeds from ACTIVE", func(t *testing.T) {
		e := &Event{Status: EventStatusActive}
		if err := e.End(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if e.Status != EventStatusEnded {
			t.Fatalf("expected ENDED, got %s", e.Status)
		}
	})

	t.Run("succeeds from SOLD_OUT", func(t *testing.T) {
		e := &Event{Status: EventStatusSoldOut}
		if err := e.End(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if e.Status != EventStatusEnded {
			t.Fatalf("expected ENDED, got %s", e.Status)
		}
	})

	t.Run("fails from DRAFT", func(t *testing.T) {
		e := &Event{Status: EventStatusDraft}
		if err := e.End(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("fails from ENDED", func(t *testing.T) {
		e := &Event{Status: EventStatusEnded}
		if err := e.End(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestEvent_CanAcceptClaims(t *testing.T) {
	cases := []struct {
		status   EventStatus
		expected bool
	}{
		{EventStatusDraft, false},
		{EventStatusActive, true},
		{EventStatusSoldOut, false},
		{EventStatusEnded, false},
	}

	for _, tc := range cases {
		e := &Event{Status: tc.status}
		if got := e.CanAcceptClaims(); got != tc.expected {
			t.Errorf("status %s: expected %v, got %v", tc.status, tc.expected, got)
		}
	}
}
