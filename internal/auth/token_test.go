package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestTokenizer_GenerateAndVerify(t *testing.T) {
	tokenizer := NewTokenizer("supersecretkey_thats_at_least_32chars", 5*time.Minute)

	t.Run("valid token verifies correctly", func(t *testing.T) {
		token, err := tokenizer.Generate("user-123", "event-456")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		userID, eventID, err := tokenizer.Verify(token)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if userID != "user-123" {
			t.Errorf("expected user-123, got %s", userID)
		}
		if eventID != "event-456" {
			t.Errorf("expected event-456, got %s", eventID)
		}
	})

	t.Run("tampered payload is rejected", func(t *testing.T) {
		token, _ := tokenizer.Generate("user-123", "event-456")

		// Replace the payload portion with a different one
		parts := strings.SplitN(token, ".", 2)
		fakePayload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"attacker","event_id":"event-456"}`))
		tampered := fakePayload + "." + parts[1]

		_, _, err := tokenizer.Verify(tampered)
		if err != ErrTokenInvalid {
			t.Fatalf("expected ErrTokenInvalid, got %v", err)
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		shortTokenizer := NewTokenizer("supersecretkey_thats_at_least_32chars", -time.Second)
		token, _ := shortTokenizer.Generate("user-123", "event-456")

		_, _, err := tokenizer.Verify(token)
		if err != ErrTokenExpired {
			t.Fatalf("expected ErrTokenExpired, got %v", err)
		}
	})

	t.Run("malformed token is rejected", func(t *testing.T) {
		_, _, err := tokenizer.Verify("notavalidtoken")
		if err != ErrTokenInvalid {
			t.Fatalf("expected ErrTokenInvalid, got %v", err)
		}
	})

	t.Run("token signed with different secret is rejected", func(t *testing.T) {
		otherTokenizer := NewTokenizer("completely_different_secret_key_xyz", 5*time.Minute)
		token, _ := otherTokenizer.Generate("user-123", "event-456")

		_, _, err := tokenizer.Verify(token)
		if err != ErrTokenInvalid {
			t.Fatalf("expected ErrTokenInvalid, got %v", err)
		}
	})
}
