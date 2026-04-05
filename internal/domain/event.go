package domain

import "time"

type EventStatus string

const (
	EventStatusDraft   EventStatus = "DRAFT"
	EventStatusActive  EventStatus = "ACTIVE"
	EventStatusSoldOut EventStatus = "SOLD_OUT"
	EventStatusEnded   EventStatus = "ENDED"
)

type Event struct {
	ID                 string
	OrganizerID        string
	Name               string
	TotalInventory     int
	AvailableInventory int   // computed at load time, never persisted
	Price              int64 // in kobo (smallest NGN unit)
	Status             EventStatus
	SaleStart          time.Time
	SaleEnd            time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CanAcceptClaims returns whether this event is in a state that
// allows new claims to be created against it.
func (e *Event) CanAcceptClaims() bool {
	return e.Status == EventStatusActive
}

// Activate transitions the event from DRAFT to ACTIVE.
func (e *Event) Activate() error {
	if e.Status != EventStatusDraft {
		return ErrInvalidTransition
	}
	e.Status = EventStatusActive
	e.UpdatedAt = time.Now()
	return nil
}

// OnInventoryDepleted transitions ACTIVE to SOLD_OUT when
// available inventory hits zero. Called by service layer after
// a confirmed claim decrements the count.
func (e *Event) OnInventoryDepleted() error {
	if e.Status != EventStatusActive {
		return ErrInvalidTransition
	}
	if e.AvailableInventory > 0 {
		return nil // not depleted yet, nothing to do
	}
	e.Status = EventStatusSoldOut
	e.UpdatedAt = time.Now()
	return nil
}

// End transitions the event to ENDED from either ACTIVE or SOLD_OUT.
func (e *Event) End() error {
	if e.Status != EventStatusActive && e.Status != EventStatusSoldOut {
		return ErrInvalidTransition
	}
	e.Status = EventStatusEnded
	e.UpdatedAt = time.Now()
	return nil
}
