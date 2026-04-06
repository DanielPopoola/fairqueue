package domain

import "errors"

var (
	// Event errors
	ErrEventNotActive    = errors.New("event is not active")
	ErrEventSoldOut      = errors.New("event is sold out")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrEventNotFound     = errors.New("event not found")

	// Claim errors
	ErrClaimNotFound     = errors.New("claim not found")
	ErrAlreadyClaimed    = errors.New("user has already has claim for this event")
	ErrClaimExpired      = errors.New("claim has expired")
	ErrClaimNotClaimable = errors.New("claim is not in a claimable state")

	// Queue errors
	ErrQueueEntryNotFound = errors.New("queue entry not found")
	ErrAlreadyInQueue     = errors.New("user is already in queue for this event")
	ErrQueueEntryExpired  = errors.New("queue entry has expired")
	ErrNotAdmitted        = errors.New("user has not been admitted from queue")

	// Payment errors
	ErrPaymentNotFound        = errors.New("payment not found")
	ErrPaymentNotReconcilable = errors.New("payment is not in a reconcilable state")
)
