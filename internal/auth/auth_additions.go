package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidPassword = errors.New("invalid password")
)

// Argon2Params are the memory-hardness parameters for argon2id.
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultArgon2Params = Argon2Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// HashPassword produces an argon2id hash of the password.
// The salt is randomly generated and embedded in the returned string
// so the hash is self-contained — no need to store the salt separately.
//
// Format: $argon2id$v=19$m=65536,t=3,p=2$<salt_b64>$<hash_b64>
func HashPassword(password string) (string, error) {
	p := DefaultArgon2Params

	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		p.Iterations,
		p.Memory,
		p.Parallelism,
		p.KeyLength,
	)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.Memory, p.Iterations, p.Parallelism,
		saltB64, hashB64,
	)
	return encoded, nil
}

// CheckPassword verifies a plaintext password against a stored argon2id hash.
// Returns nil on match, ErrInvalidPassword on mismatch.
func CheckPassword(encodedHash, password string) error {
	p, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return fmt.Errorf("decoding hash: %w", err)
	}

	candidate := argon2.IDKey(
		[]byte(password),
		salt,
		p.Iterations,
		p.Memory,
		p.Parallelism,
		p.KeyLength,
	)

	// Constant-time comparison prevents timing attacks
	if !hmac.Equal(hash, candidate) {
		return ErrInvalidPassword
	}
	return nil
}

// decodeHash parses the encoded argon2id hash string back into its components.
func decodeHash(encoded string) (params Argon2Params, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return Argon2Params{}, nil, nil, errors.New("invalid hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("parsing version: %w", err)
	}
	if version != argon2.Version {
		return Argon2Params{}, nil, nil, fmt.Errorf("incompatible argon2 version: %d", version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("parsing params: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("decoding salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("decoding hash: %w", err)
	}

	params.KeyLength = uint32(len(hash)) //nolint:gosec // argon2 hashes are small making overflow impossible
	return params, salt, hash, nil
}

// organizerClaims is the payload inside an organizer JWT.
type organizerClaims struct {
	OrganizerID string    `json:"organizer_id"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// OrganizerTokenizer issues and verifies long-lived organizer session JWTs.
// Verify returns only the organizer ID
type OrganizerTokenizer struct {
	secret []byte
	ttl    time.Duration
}

func NewOrganizerTokenizer(secret string, ttl time.Duration) *OrganizerTokenizer {
	return &OrganizerTokenizer{secret: []byte(secret), ttl: ttl}
}

func (t *OrganizerTokenizer) Generate(organizerID string) (string, error) {
	payload := organizerClaims{
		OrganizerID: organizerID,
		ExpiresAt:   time.Now().Add(t.ttl),
	}
	return signPayload(t.secret, payload)
}

// Verify validates the token and returns the organizer ID.
func (t *OrganizerTokenizer) Verify(token string) (string, error) {
	var payload organizerClaims
	if err := verifyPayload(t.secret, token, &payload); err != nil {
		return "", err
	}
	return payload.OrganizerID, nil
}

// customerClaims is the payload inside a customer session JWT.
type customerClaims struct {
	CustomerID string    `json:"customer_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CustomerTokenizer issues and verifies customer session JWTs.
// It also vends short-lived admission tokens via GenerateAdmission,
// delegating to the existing Tokenizer under the hood.
type CustomerTokenizer struct {
	secret    []byte
	ttl       time.Duration
	admission *Tokenizer // reuses existing admission token logic
}

func NewCustomerTokenizer(secret string, ttl, admissionTTL time.Duration) *CustomerTokenizer {
	return &CustomerTokenizer{
		secret:    []byte(secret),
		ttl:       ttl,
		admission: NewTokenizer(secret, admissionTTL),
	}
}

func (t *CustomerTokenizer) Generate(customerID string) (string, error) {
	payload := customerClaims{
		CustomerID: customerID,
		ExpiresAt:  time.Now().Add(t.ttl),
	}
	return signPayload(t.secret, payload)
}

// Verify validates the token and returns the customer ID.
func (t *CustomerTokenizer) Verify(token string) (string, error) {
	var payload customerClaims
	if err := verifyPayload(t.secret, token, &payload); err != nil {
		return "", err
	}
	return payload.CustomerID, nil
}

// GenerateAdmission issues a short-lived, event-scoped admission token.
// Called by GetQueuePosition when a customer polls for their status
// after being admitted — the fallback for customers who missed the
// WebSocket push.
func (t *CustomerTokenizer) GenerateAdmission(customerID, eventID string) (string, error) {
	return t.admission.Generate(customerID, eventID)
}

// ── Shared HMAC signing helpers ───────────────────────────────
// These are package-private so only tokenizers in this package use them.
// External callers use the tokenizer types directly.
func signPayload(secret []byte, payload any) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshalling token payload: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := computeHMAC(secret, encodedPayload)

	return encodedPayload + "." + sig, nil
}

func verifyPayload(secret []byte, token string, dst any) error {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return ErrTokenInvalid
	}

	encodedPayload, sig := parts[0], parts[1]

	expected := computeHMAC(secret, encodedPayload)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return ErrTokenInvalid
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return ErrTokenInvalid
	}

	if err := json.Unmarshal(payloadBytes, dst); err != nil {
		return ErrTokenInvalid
	}

	// Direct expiry check via type switch — cleaner than reflection
	// since we only have two claim types.
	switch v := dst.(type) {
	case *organizerClaims:
		if time.Now().After(v.ExpiresAt) {
			return ErrTokenExpired
		}
	case *customerClaims:
		if time.Now().After(v.ExpiresAt) {
			return ErrTokenExpired
		}
	}

	return nil
}

func computeHMAC(secret []byte, data string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
