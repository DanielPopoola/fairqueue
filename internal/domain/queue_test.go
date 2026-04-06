package domain

import (
	"testing"
	"time"
)

func TestQueueEntry_IsExpired(t *testing.T) {
	t.Run("false for fresh WAITING entry", func(t *testing.T) {
		q := &QueueEntry{Status: QueueEntryStatusWaiting, JoinedAt: time.Now()}
		if q.IsExpired() {
			t.Fatal("expected not expired")
		}
	})

	t.Run("true for WAITING entry beyond TTL", func(t *testing.T) {
		q := &QueueEntry{
			Status:   QueueEntryStatusWaiting,
			JoinedAt: time.Now().Add(-(QueueEntryTTL + time.Second)),
		}
		if !q.IsExpired() {
			t.Fatal("expected expired")
		}
	})

	t.Run("false for fresh ADMITTED entry", func(t *testing.T) {
		now := time.Now()
		q := &QueueEntry{
			Status:     QueueEntryStatusAdmitted,
			JoinedAt:   now.Add(-time.Hour),
			AdmittedAt: &now,
		}
		if q.IsExpired() {
			t.Fatal("expected not expired")
		}
	})

	t.Run("true for ADMITTED entry beyond admission window", func(t *testing.T) {
		admittedAt := time.Now().Add(-(AdmissionWindowTTL + time.Second))
		q := &QueueEntry{
			Status:     QueueEntryStatusAdmitted,
			JoinedAt:   admittedAt.Add(-time.Hour),
			AdmittedAt: &admittedAt,
		}
		if !q.IsExpired() {
			t.Fatal("expected expired")
		}
	})

	t.Run("false for COMPLETED regardless of age", func(t *testing.T) {
		q := &QueueEntry{
			Status:   QueueEntryStatusCompleted,
			JoinedAt: time.Now().Add(-(QueueEntryTTL + time.Hour)),
		}
		if q.IsExpired() {
			t.Fatal("completed entries should never expire")
		}
	})
}

func TestQueueEntry_Admit(t *testing.T) {
	t.Run("succeeds for fresh WAITING entry", func(t *testing.T) {
		q := &QueueEntry{Status: QueueEntryStatusWaiting, JoinedAt: time.Now()}
		if err := q.Admit(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if q.Status != QueueEntryStatusAdmitted {
			t.Fatalf("expected ADMITTED, got %s", q.Status)
		}
		if q.AdmittedAt == nil {
			t.Fatal("expected AdmittedAt to be set")
		}
	})

	t.Run("fails for expired WAITING entry", func(t *testing.T) {
		q := &QueueEntry{
			Status:   QueueEntryStatusWaiting,
			JoinedAt: time.Now().Add(-(QueueEntryTTL + time.Second)),
		}
		if err := q.Admit(); err != ErrQueueEntryExpired {
			t.Fatalf("expected ErrQueueEntryExpired, got %v", err)
		}
	})

	t.Run("fails for already ADMITTED", func(t *testing.T) {
		now := time.Now()
		q := &QueueEntry{Status: QueueEntryStatusAdmitted, AdmittedAt: &now}
		if err := q.Admit(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestQueueEntry_Complete(t *testing.T) {
	t.Run("succeeds for ADMITTED entry", func(t *testing.T) {
		now := time.Now()
		q := &QueueEntry{Status: QueueEntryStatusAdmitted, AdmittedAt: &now}
		if err := q.Complete(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if q.Status != QueueEntryStatusCompleted {
			t.Fatalf("expected COMPLETED, got %s", q.Status)
		}
	})

	t.Run("fails for WAITING entry", func(t *testing.T) {
		q := &QueueEntry{Status: QueueEntryStatusWaiting}
		if err := q.Complete(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestQueueEntry_Abandon(t *testing.T) {
	t.Run("succeeds for WAITING entry", func(t *testing.T) {
		q := &QueueEntry{Status: QueueEntryStatusWaiting, JoinedAt: time.Now()}
		if err := q.Abandon(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if q.Status != QueueEntryStatusAbandoned {
			t.Fatalf("expected ABANDONED, got %s", q.Status)
		}
	})

	t.Run("fails for ADMITTED entry", func(t *testing.T) {
		now := time.Now()
		q := &QueueEntry{Status: QueueEntryStatusAdmitted, AdmittedAt: &now}
		if err := q.Abandon(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

func TestQueueEntry_Expire(t *testing.T) {
	t.Run("succeeds for WAITING entry", func(t *testing.T) {
		q := &QueueEntry{Status: QueueEntryStatusWaiting}
		if err := q.Expire(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if q.Status != QueueEntryStatusExpired {
			t.Fatalf("expected EXPIRED, got %s", q.Status)
		}
	})

	t.Run("succeeds for ADMITTED entry", func(t *testing.T) {
		now := time.Now()
		q := &QueueEntry{Status: QueueEntryStatusAdmitted, AdmittedAt: &now}
		if err := q.Expire(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if q.Status != QueueEntryStatusExpired {
			t.Fatalf("expected EXPIRED, got %s", q.Status)
		}
	})

	t.Run("fails for COMPLETED entry", func(t *testing.T) {
		q := &QueueEntry{Status: QueueEntryStatusCompleted}
		if err := q.Expire(); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}
