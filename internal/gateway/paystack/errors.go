// Package paystack implements the gateway.PaymentGateway interface
// using the Paystack payment provider.
package paystack

import "errors"

var (
	// ErrPermanent is returned for errors Paystack explicitly rejects.
	// Examples: invalid amount, invalid email, duplicate reference conflict.
	// These should NOT be retried.
	ErrPermanent = errors.New("permanent paystack error")

	// ErrTransient is returned for errors that may resolve on retry.
	// Examples: network timeouts, 5xx server errors.
	ErrTransient = errors.New("transient paystack error")
)

// IsTransient returns true if the error should be retried.
func IsTransient(err error) bool {
	return errors.Is(err, ErrTransient)
}
