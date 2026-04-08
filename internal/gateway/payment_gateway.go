// Package gateway defines interfaces for external payment providers.
package gateway

import "context"

// PaymentGateway defines the contract for payment provider integrations.
//
//go:generate mockery --name PaymentGateway --output ./mocks --outpkg mock
type PaymentGateway interface {
	// InitializeTransaction creates a new transaction and returns
	// the URL the customer uses to complete payment.
	InitializeTransaction(ctx context.Context, req InitializeRequest) (*InitializeResponse, error)

	// VerifyTransaction checks the current status of a transaction
	// by its reference. Used by the reconciliation worker.
	VerifyTransaction(ctx context.Context, reference string) (*VerifyResponse, error)

	// VerifyWebhookSignature confirms a webhook payload came from
	// the payment provider and has not been tampered with.
	VerifyWebhookSignature(payload []byte, signature string) bool
}

// InitializeRequest for initializing requests to Paystack
type InitializeRequest struct {
	Email      string
	AmountKobo int64
	Reference  string
}

// InitializeResponse is the response from Paystack upon successful implementation
type InitializeResponse struct {
	AuthorizationURL string
	Reference        string
}

// VerifyResponse is the Paystack result of a
// transaction status check.
type VerifyResponse struct {
	Status          string // "success", "failed", "abandoned"
	GatewayResponse string // human-readable reason from provider
	Reference       string
}

// WebhookPayload is the Paystack representation of
// an incoming webhook event.
type WebhookPayload struct {
	Event string
	Data  WebhookData
}

// WebhookData contains the transaction details from the webhook.
type WebhookData struct {
	Reference       string
	Status          string
	GatewayResponse string
}
