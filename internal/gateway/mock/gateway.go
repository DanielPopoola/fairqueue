// Package mock provides runtime-safe payment gateway doubles.
package mock

import (
	"context"
	"fmt"

	"github.com/DanielPopoola/fairqueue/internal/gateway"
)

// Gateway is a deterministic payment gateway used for load testing.
//
// It avoids network calls and returns immediate success responses so
// throughput measurements reflect FairQueue's own bottlenecks.
type Gateway struct{}

func NewGateway() *Gateway {
	return &Gateway{}
}

// InitializeTransaction returns an immediate authorization response.
func (g *Gateway) InitializeTransaction(ctx context.Context, req gateway.InitializeRequest) (*gateway.InitializeResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &gateway.InitializeResponse{
		AuthorizationURL: fmt.Sprintf("https://mock-gateway.local/pay/%s", req.Reference),
		Reference:        req.Reference,
	}, nil
}

// VerifyTransaction always returns a successful verification.
func (g *Gateway) VerifyTransaction(ctx context.Context, reference string) (*gateway.VerifyResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &gateway.VerifyResponse{
		Status:          "success",
		GatewayResponse: "load test mock success",
		Reference:       reference,
	}, nil
}

// VerifyWebhookSignature trusts all signatures in load test mode.
func (g *Gateway) VerifyWebhookSignature(_ []byte, _ string) bool {
	return true
}
