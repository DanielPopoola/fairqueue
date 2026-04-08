package domain

import "errors"

var (
	// Organizer errors
	ErrOrganizerNotFound = errors.New("organizer not found")

	// Customer errors
	ErrCustomerNotFound = errors.New("customer not found")

	// Event errors
	ErrEventNotFound     = errors.New("event not found")
	ErrEventNotActive    = errors.New("event is not active")
	ErrEventSoldOut      = errors.New("event is sold out")
	ErrInvalidTransition = errors.New("invalid state transition")

	// Claim errors
	ErrClaimNotFound     = errors.New("claim not found")
	ErrAlreadyClaimed    = errors.New("customer already has a claim for this event")
	ErrClaimExpired      = errors.New("claim has expired")
	ErrClaimNotClaimable = errors.New("claim is not in a claimable state")

	// Queue errors
	ErrQueueEntryNotFound = errors.New("queue entry not found")
	ErrAlreadyInQueue     = errors.New("customer is already in queue for this event")
	ErrQueueEntryExpired  = errors.New("queue entry has expired")
	ErrNotAdmitted        = errors.New("customer has not been admitted from queue")

	// Payment errors
	ErrPaymentNotFound        = errors.New("payment not found")
	ErrPaymentAlreadyMade     = errors.New("payment already exists")
	ErrPaymentNotReconcilable = errors.New("payment is not in a reconcilable state")
)
