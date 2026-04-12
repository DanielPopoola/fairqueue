package api

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/metrics"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/go-chi/chi/v5"
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

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.status)

		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, routePattern, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
