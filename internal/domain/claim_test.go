package domain

import (
	"testing"
	"time"
)

func TestClaim_IsExpired(t *testing.T) {
	t.Run("false for fresh CLAIMED", func(t *testing.T) {
		c := &Claim{Status: ClaimStatusClaimed, CreatedAt: time.Now()}
		if c.IsExpired() {
			t.Fatal("expected not expired")
		}
	})

	t.Run("true for old CLAIMED beyond TTL", func(t *testing.T) {
		c := &Claim{
			Status:    ClaimStatusClaimed,
			CreatedAt: time.Now().Add(-(ClaimTTL + time.Second)),
		}
		if !c.IsExpired() {
			t.Fatal("expected expired")
		}
	})

	t.Run("false for CONFIRMED regardless of age", func(t *testing.T) {
		c := &Claim{
			Status:    ClaimStatusConfirmed,
			CreatedAt: time.Now().Add(-(ClaimTTL + time.Hour)),
		}
		if c.IsExpired() {
			t.Fatal("confirmed claims should never expire")
		}
	})

	t.Run("false for RELEASED regardless of age", func(t *testing.T) {
		c := &Claim{
			Status:    ClaimStatusReleased,
			CreatedAt: time.Now().Add(-(ClaimTTL + time.Hour)),
		}
		if c.IsExpired() {
			t.Fatal("released claims should never expire")
		}
	})
}

func TestClaim_Confirm(t *testing.T) {
	t.Run("succeeds for fresh CLAIMED", func(t *testing.T) {
		c := &Claim{Status: ClaimStatusClaimed, CreatedAt: time.Now()}
		if err := c.Confirm(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c.Status != ClaimStatusConfirmed {
			t.Fatalf("expected CONFIRMED, got %s", c.Status)
		}
	})

	t.Run("fails for expired CLAIMED", func(t *testing.T) {
		c := &Claim{
			Status:    ClaimStatusClaimed,
			CreatedAt: time.Now().Add(-(ClaimTTL + time.Second)),
		}
		if err := c.Confirm(); err != ErrClaimExpired {
			t.Fatalf("expected ErrClaimExpired, got %v", err)
		}
	})

	t.Run("fails for already CONFIRMED", func(t *testing.T) {
		c := &Claim{Status: ClaimStatusConfirmed, CreatedAt: time.Now()}
		if err := c.Confirm(); err != ErrClaimNotClaimable {
			t.Fatalf("expected ErrClaimNotClaimable, got %v", err)
		}
	})

	t.Run("fails for RELEASED", func(t *testing.T) {
		c := &Claim{Status: ClaimStatusReleased, CreatedAt: time.Now()}
		if err := c.Confirm(); err != ErrClaimNotClaimable {
			t.Fatalf("expected ErrClaimNotClaimable, got %v", err)
		}
	})
}

func TestClaim_Release(t *testing.T) {
	t.Run("succeeds for CLAIMED", func(t *testing.T) {
		c := &Claim{Status: ClaimStatusClaimed, CreatedAt: time.Now()}
		if err := c.Release(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c.Status != ClaimStatusReleased {
			t.Fatalf("expected RELEASED, got %s", c.Status)
		}
	})

	t.Run("fails for CONFIRMED", func(t *testing.T) {
		c := &Claim{Status: ClaimStatusConfirmed, CreatedAt: time.Now()}
		if err := c.Release(); err != ErrClaimNotClaimable {
			t.Fatalf("expected ErrClaimNotClaimable, got %v", err)
		}
	})

	t.Run("fails for already RELEASED", func(t *testing.T) {
		c := &Claim{Status: ClaimStatusReleased, CreatedAt: time.Now()}
		if err := c.Release(); err != ErrClaimNotClaimable {
			t.Fatalf("expected ErrClaimNotClaimable, got %v", err)
		}
	})
}
