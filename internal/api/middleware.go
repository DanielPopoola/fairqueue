package api

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
)

const otpTTL = 10 * time.Minute

// OrganizerAuthMiddleware validates the Bearer JWT and injects the
// organizer ID into the request context.
func OrganizerAuthMiddleware(tokenizer *auth.OrganizerTokenizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			organizerID, err := tokenizer.Verify(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), organizerIDKey, organizerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CustomerAuthMiddleware validates the Bearer JWT and injects the
// customer ID into the request context.
func CustomerAuthMiddleware(tokenizer *auth.CustomerTokenizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			customerID, err := tokenizer.Verify(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), customerIDKey, customerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(header, "Bearer "), true
}

// ── OTP store ─────────────────────────────────────────────────

// OTPStore manages short-lived OTP codes in Redis.
// Key: otp:{email}  Value: 6-digit code  TTL: 10 minutes
type OTPStore struct {
	client *redisstore.Client
}

func NewOTPStore(client *redisstore.Client) *OTPStore {
	return &OTPStore{client: client}
}

func (s *OTPStore) Save(ctx context.Context, email, code string) error {
	return s.client.SetEX(ctx, otpKey(email), code, otpTTL)
}

func (s *OTPStore) Verify(ctx context.Context, email, code string) error {
	stored, err := s.client.Get(ctx, otpKey(email))
	if err != nil {
		return fmt.Errorf("OTP not found or expired: %w", err)
	}
	if stored != code {
		return errors.New("invalid OTP")
	}
	// Single-use — delete after successful verification
	_ = s.client.Del(ctx, otpKey(email)) //nolint:errcheck // error check not necessary here
	return nil
}

func otpKey(email string) string {
	return "otp:" + email
}

// generateOTP returns a cryptographically random 6-digit string.
func generateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating OTP: %w", err)
	}
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}
