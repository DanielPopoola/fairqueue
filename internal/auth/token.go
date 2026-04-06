// Package auth handles admission token generation and verification.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token has expired")
)

// claims is the payload encoded inside the token.
// Unexported because nothing outside this package
// should construct one directly.
type claims struct {
	CustomerID string    `json:"customer_id"`
	EventID    string    `json:"event_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Tokenizer generates and verifies admission tokens.
type Tokenizer struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenizer creates a Tokenizer with the given secret and TTL.
func NewTokenizer(secret string, ttl time.Duration) *Tokenizer {
	return &Tokenizer{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Generate creates a signed admission token for the given user and event.
func (t *Tokenizer) Generate(customerID, eventID string) (string, error) {
	payload := claims{
		CustomerID: customerID,
		EventID:    eventID,
		ExpiresAt:  time.Now().Add(t.ttl),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshalling token payload: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := t.sign(encodedPayload)

	return encodedPayload + "." + sig, nil
}

// Verify parses and validates a token, returning the user and event IDs
// if the token is valid and unexpired.
func (t *Tokenizer) Verify(token string) (customerID, eventID string, err error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", "", ErrTokenInvalid
	}

	encodedPayload, sig := parts[0], parts[1]

	// Recompute signature and compare — this is the cryptographic check.
	// If someone tampered with the payload, the signatures won't match.
	expected := t.sign(encodedPayload)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", "", ErrTokenInvalid
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return "", "", ErrTokenInvalid
	}

	var payload claims
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", "", ErrTokenInvalid
	}

	if time.Now().After(payload.ExpiresAt) {
		return "", "", ErrTokenExpired
	}

	return payload.CustomerID, payload.EventID, nil
}

// sign computes an HMAC-SHA256 signature of the given data.
// Private because callers should use Generate and Verify,
// never sign directly.
func (t *Tokenizer) sign(data string) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
