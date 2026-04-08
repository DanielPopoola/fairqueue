package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/gateway"
	"github.com/DanielPopoola/fairqueue/internal/infra/retry"
)

// Client implements gateway.PaymentGateway using the Paystack API.
type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
	retryCfg   retry.Config
}

// NewClient creates a Paystack client.
// Verify it satisfies the interface at compile time.
var _ gateway.PaymentGateway = (*Client)(nil)

func NewClient(secretKey, baseURL string, cfg retry.Config) *Client {
	return &Client{
		secretKey: secretKey,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryCfg: cfg,
	}
}

// InitializeTransaction implements gateway.PaymentGateway.
func (c *Client) InitializeTransaction(
	ctx context.Context,
	req gateway.InitializeRequest,
) (*gateway.InitializeResponse, error) {
	return retry.Do(ctx, c.retryCfg, IsTransient, func() (*gateway.InitializeResponse, error) {
		return c.doInitialize(ctx, req)
	})
}

// VerifyTransaction implements gateway.PaymentGateway.
func (c *Client) VerifyTransaction(
	ctx context.Context,
	reference string,
) (*gateway.VerifyResponse, error) {
	return retry.Do(ctx, c.retryCfg, IsTransient, func() (*gateway.VerifyResponse, error) {
		return c.doVerify(ctx, reference)
	})
}

// VerifyWebhookSignature implements gateway.PaymentGateway.
func (c *Client) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(c.secretKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (c *Client) doInitialize(ctx context.Context, req gateway.InitializeRequest) (*gateway.InitializeResponse, error) {
	body, err := json.Marshal(map[string]any{
		"email":     req.Email,
		"amount":    req.AmountKobo,
		"reference": req.Reference,
	})
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/transaction/initialize",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: creating request: %w", ErrTransient, err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTransient, err)
	}
	defer resp.Body.Close() //nolint:errcheck // Closing the response body; error can be ignored here.

	return c.parseInitializeResponse(resp)
}

func (c *Client) doVerify(ctx context.Context, reference string) (*gateway.VerifyResponse, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/transaction/verify/"+reference,
		http.NoBody,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: creating request: %w", ErrTransient, err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTransient, err)
	}
	defer resp.Body.Close() //nolint:errcheck // Closing the response body; error can be ignored here.

	return c.parseVerifyResponse(resp)
}

func (c *Client) parseInitializeResponse(resp *http.Response) (*gateway.InitializeResponse, error) {
	var envelope struct {
		apiResponse
		Data initializeData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("%w: decoding response", ErrTransient)
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: server error %d", ErrTransient, resp.StatusCode)
	}

	if resp.StatusCode >= 400 || !envelope.Status {
		return nil, fmt.Errorf("%w: %s", ErrPermanent, envelope.Message)
	}

	return &gateway.InitializeResponse{
		AuthorizationURL: envelope.Data.AuthorizationURL,
		Reference:        envelope.Data.Reference,
	}, nil
}

func (c *Client) parseVerifyResponse(resp *http.Response) (*gateway.VerifyResponse, error) {
	var envelope struct {
		apiResponse
		Data verifyData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("%w: decoding response", ErrTransient)
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: server error %d", ErrTransient, resp.StatusCode)
	}

	if resp.StatusCode >= 400 || !envelope.Status {
		return nil, fmt.Errorf("%w: %s", ErrPermanent, envelope.Message)
	}

	return &gateway.VerifyResponse{
		Status:          envelope.Data.Status,
		GatewayResponse: envelope.Data.GatewayResponse,
		Reference:       envelope.Data.Reference,
	}, nil
}
