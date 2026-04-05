package domain

import "time"

type ClaimStatus string

const (
	ClaimStatusClaimed   ClaimStatus = "CLAIMED"
	ClaimStatusConfirmed ClaimStatus = "CONFIRMED"
	ClaimStatusReleased  ClaimStatus = "RELEASED"
)

const ClaimTTL = 10 * time.Minute

type Claim struct {
	ID        string
	EventID   string
	UserID    string
	Status    ClaimStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsExpired returns true if the claim has been in CLAIMED state
// longer than the TTL. Only CLAIMED claims can expire — CONFIRMED
// claims are permanent.
func (c *Claim) IsExpired() bool {
	if c.Status != ClaimStatusClaimed {
		return false
	}
	return time.Since(c.CreatedAt) > ClaimTTL
}

// Confirm transitions the claim from CLAIMED to CONFIRMED.
// Returns an error if the claim is expired or in the wrong state.
func (c *Claim) Confirm() error {
	if c.Status != ClaimStatusClaimed {
		return ErrClaimNotClaimable
	}
	if c.IsExpired() {
		return ErrClaimExpired
	}
	c.Status = ClaimStatusConfirmed
	c.UpdatedAt = time.Now()
	return nil
}

// Release transitions the claim from CLAIMED to RELEASED.
// Used for both explicit releases and expiry cleanup.
func (c *Claim) Release() error {
	if c.Status != ClaimStatusClaimed {
		return ErrClaimNotClaimable
	}
	c.Status = ClaimStatusReleased
	c.UpdatedAt = time.Now()
	return nil
}
