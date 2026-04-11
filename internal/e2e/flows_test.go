package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/gateway"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
)

// ─────────────────────────────────────────────────────────────
// Flow 1: Full happy path
// Organizer creates event → customer joins queue → admitted →
// claims ticket → pays → payment confirmed
// ─────────────────────────────────────────────────────────────

func TestFlow_HappyPath(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	client := sharedEnv.client

	// ── Step 1: Organizer logs in ─────────────────────────────
	email, password := seedOrganizer(testCtx, t, sharedEnv.pool)

	resp := client.POST(t, "/auth/organizer/login", map[string]any{
		"email":    email,
		"password": password,
	})
	assertStatus(t, resp, http.StatusOK)

	type authResp struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
	}
	loginResp := decodeBody[authResp](t, resp)
	if loginResp.Token == "" {
		t.Fatal("expected organizer token, got empty string")
	}

	orgClient := client.WithToken(loginResp.Token)

	// ── Step 2: Create event ──────────────────────────────────
	resp = orgClient.POST(t, "/events", map[string]any{
		"name":            "Burna Boy Live in Lagos",
		"total_inventory": 100,
		"price":           25000,
		"sale_start":      time.Now().Add(-time.Hour).Format(time.RFC3339),
		"sale_end":        time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	assertStatus(t, resp, http.StatusCreated)

	type eventResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	createResp := decodeBody[eventResp](t, resp)
	eventID := createResp.Data.ID
	if createResp.Data.Status != "DRAFT" {
		t.Fatalf("expected DRAFT, got %s", createResp.Data.Status)
	}

	// ── Step 3: Activate event ────────────────────────────────
	resp = orgClient.PUT(t, "/events/"+eventID+"/activate")
	assertStatus(t, resp, http.StatusOK)

	activated := decodeBody[eventResp](t, resp)
	if activated.Data.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE, got %s", activated.Data.Status)
	}

	// ── Step 4: Customer requests OTP ────────────────────────
	custEmail := fmt.Sprintf("customer-%d@test.com", time.Now().UnixNano())
	resp = client.POST(t, "/auth/customer/otp/request", map[string]any{
		"email": custEmail,
	})
	assertStatus(t, resp, http.StatusOK)

	// Retrieve OTP directly from Redis (bypasses email in tests)
	otpKey := "otp:" + custEmail
	otp, err := sharedEnv.rdb.Get(testCtx, otpKey).Result()
	if err != nil {
		t.Fatalf("reading OTP from redis: %v", err)
	}

	// ── Step 5: Customer verifies OTP ────────────────────────
	resp = client.POST(t, "/auth/customer/otp/verify", map[string]any{
		"email": custEmail,
		"otp":   otp,
	})
	assertStatus(t, resp, http.StatusOK)

	type custAuthResp struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
	}
	custResp := decodeBody[custAuthResp](t, resp)
	custClient := client.WithToken(custResp.Token)

	// ── Step 6: Customer joins queue ─────────────────────────
	resp = custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)

	type queueJoinResp struct {
		Success  bool  `json:"success"`
		Position int64 `json:"position"`
	}
	joinResp := decodeBody[queueJoinResp](t, resp)
	if joinResp.Position != 1 {
		t.Fatalf("expected position 1, got %d", joinResp.Position)
	}

	// ── Step 7: Admission worker runs ────────────────────────
	// Warm the inventory cache first (normally done by startup recovery)
	db := postgres.NewDB(sharedEnv.pool)
	event, err := postgres.NewEventStore(db).GetByID(testCtx, eventID)
	if err != nil {
		t.Fatalf("fetching event: %v", err)
	}
	rc, _ := redisstore.NewClient(testCtx, sharedEnv.rdb)
	redisstore.NewInventoryStore(rc).Set(testCtx, eventID, int64(event.TotalInventory))

	if err := sharedEnv.admissionWorker.Run(testCtx); err != nil {
		t.Fatalf("admission worker: %v", err)
	}

	// ── Step 8: Customer polls position — expects ADMITTED ────
	resp = custClient.GET(t, "/events/"+eventID+"/queue/position")
	assertStatus(t, resp, http.StatusOK)

	type positionResp struct {
		Success        bool    `json:"success"`
		Status         string  `json:"status"`
		AdmissionToken *string `json:"admission_token"`
	}
	posResp := decodeBody[positionResp](t, resp)
	if posResp.Status != "ADMITTED" {
		t.Fatalf("expected ADMITTED, got %s", posResp.Status)
	}
	if posResp.AdmissionToken == nil || *posResp.AdmissionToken == "" {
		t.Fatal("expected admission token in position response")
	}
	admissionToken := *posResp.AdmissionToken

	// ── Step 9: Customer claims ticket ───────────────────────
	resp = custClient.POST(t, "/events/"+eventID+"/claims", map[string]any{
		"admission_token": admissionToken,
	})
	assertStatus(t, resp, http.StatusCreated)

	type claimResp struct {
		Success bool   `json:"success"`
		ClaimID string `json:"claim_id"`
	}
	claimResult := decodeBody[claimResp](t, resp)
	claimID := claimResult.ClaimID
	if claimID == "" {
		t.Fatal("expected claim_id in response")
	}

	// ── Step 10: Customer initializes payment ─────────────────
	ref := fmt.Sprintf("fq-e2e-%d", time.Now().UnixNano())
	sharedEnv.gateway.On("InitializeTransaction", mock.Anything, mock.Anything).
		Return(&gateway.InitializeResponse{
			AuthorizationURL: "https://checkout.paystack.com/pay/test",
			Reference:        ref,
		}, nil).Once()

	resp = custClient.POST(t, "/claims/"+claimID+"/payments", nil)
	assertStatus(t, resp, http.StatusCreated)

	type paymentResp struct {
		Success          bool   `json:"success"`
		PaymentID        string `json:"payment_id"`
		AuthorizationURL string `json:"authorization_url"`
	}
	payResp := decodeBody[paymentResp](t, resp)
	if payResp.AuthorizationURL == "" {
		t.Fatal("expected authorization_url in response")
	}

	// ── Step 11: Paystack sends charge.success webhook ────────
	sharedEnv.gateway.On("VerifyWebhookSignature", mock.Anything, "valid-sig").Return(true)

	resp = client.POST(t, "/webhooks/paystack", webhookPayload("charge.success", ref))
	// Add signature header
	resp.Body.Close()

	// Re-send with proper signature header
	req := mustBuildWebhookRequest(t, client.baseURL+"/webhooks/paystack",
		webhookPayload("charge.success", ref), "valid-sig")
	webhookResp, err := client.http.Do(req)
	if err != nil {
		t.Fatalf("webhook request: %v", err)
	}
	assertStatus(t, webhookResp, http.StatusOK)
	webhookResp.Body.Close()

	// ── Step 12: Verify final state in Postgres ───────────────
	db = postgres.NewDB(sharedEnv.pool)
	payment, err := postgres.NewPaymentStore(db).GetByClaimID(testCtx, claimID)
	if err != nil {
		t.Fatalf("fetching payment: %v", err)
	}
	if payment.Status != domain.PaymentStatusConfirmed {
		t.Fatalf("expected payment CONFIRMED, got %s", payment.Status)
	}

	claim, err := postgres.NewClaimStore(db).GetByID(testCtx, claimID)
	if err != nil {
		t.Fatalf("fetching claim: %v", err)
	}
	if claim.Status != domain.ClaimStatusConfirmed {
		t.Fatalf("expected claim CONFIRMED, got %s", claim.Status)
	}
}

// ─────────────────────────────────────────────────────────────
// Flow 2: Sold-out transition
// Last ticket claimed → event transitions to SOLD_OUT →
// next customer gets 410
// ─────────────────────────────────────────────────────────────

func TestFlow_SoldOut(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	client := sharedEnv.client

	// Setup: organizer + event with 1 ticket
	orgClient := mustLoginOrganizer(t, client, sharedEnv.pool)
	eventID := mustCreateAndActivateEvent(t, orgClient, sharedEnv.pool, sharedEnv.rdb, 1)

	// Warm inventory
	rc, _ := redisstore.NewClient(testCtx, sharedEnv.rdb)
	redisstore.NewInventoryStore(rc).Set(testCtx, eventID, 1)

	// Two customers join queue
	cust1Token := mustCustomerAuth(t, client, sharedEnv.rdb)
	cust2Token := mustCustomerAuth(t, client, sharedEnv.rdb)

	cust1 := client.WithToken(cust1Token)
	cust2 := client.WithToken(cust2Token)

	resp := cust1.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = cust2.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Admit both
	if err := sharedEnv.admissionWorker.Run(testCtx); err != nil {
		t.Fatalf("admission worker: %v", err)
	}

	// Customer 1 claims the last ticket
	admToken1 := mustGetAdmissionToken(t, cust1, eventID)
	resp = cust1.POST(t, "/events/"+eventID+"/claims", map[string]any{
		"admission_token": admToken1,
	})
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Event should now be SOLD_OUT
	db := postgres.NewDB(sharedEnv.pool)
	event, err := postgres.NewEventStore(db).GetByID(testCtx, eventID)
	if err != nil {
		t.Fatalf("fetching event: %v", err)
	}
	if event.Status != domain.EventStatusSoldOut {
		t.Fatalf("expected SOLD_OUT, got %s", event.Status)
	}

	// Customer 2 tries to claim — should get 410
	admToken2 := mustGetAdmissionToken(t, cust2, eventID)
	resp = cust2.POST(t, "/events/"+eventID+"/claims", map[string]any{
		"admission_token": admToken2,
	})
	assertStatus(t, resp, http.StatusGone)
	resp.Body.Close()
}

// ─────────────────────────────────────────────────────────────
// Flow 3: Claim expiry restores inventory
// ─────────────────────────────────────────────────────────────

func TestFlow_ClaimExpiry_RestoresInventory(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	client := sharedEnv.client
	orgClient := mustLoginOrganizer(t, client, sharedEnv.pool)
	eventID := mustCreateAndActivateEvent(t, orgClient, sharedEnv.pool, sharedEnv.rdb, 10)

	rc, _ := redisstore.NewClient(testCtx, sharedEnv.rdb)
	invStore := redisstore.NewInventoryStore(rc)
	invStore.Set(testCtx, eventID, 10)

	// Customer joins, gets admitted, claims
	custToken := mustCustomerAuth(t, client, sharedEnv.rdb)
	custClient := client.WithToken(custToken)

	resp := custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	if err := sharedEnv.admissionWorker.Run(testCtx); err != nil {
		t.Fatalf("admission worker: %v", err)
	}

	admToken := mustGetAdmissionToken(t, custClient, eventID)
	resp = custClient.POST(t, "/events/"+eventID+"/claims", map[string]any{
		"admission_token": admToken,
	})
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Inventory should be 9 now
	count, _ := invStore.Get(testCtx, eventID)
	if count != 9 {
		t.Fatalf("expected inventory 9 after claim, got %d", count)
	}

	// Backdate the claim so the expiry worker picks it up
	_, err := sharedEnv.pool.Exec(testCtx,
		`UPDATE claims SET created_at = $1, updated_at = $1 WHERE status = 'CLAIMED' AND event_id = $2`,
		time.Now().Add(-(domain.ClaimTTL + time.Minute)), eventID,
	)
	if err != nil {
		t.Fatalf("backdating claim: %v", err)
	}

	// Run expiry worker
	if err := sharedEnv.expiryWorker.Run(testCtx); err != nil {
		t.Fatalf("expiry worker: %v", err)
	}

	// Inventory should be restored to 10
	count, _ = invStore.Get(testCtx, eventID)
	if count != 10 {
		t.Fatalf("expected inventory 10 after expiry, got %d", count)
	}

	// Verify claim is RELEASED in Postgres
	db := postgres.NewDB(sharedEnv.pool)
	expired, err := postgres.NewClaimStore(db).GetExpiredClaims(testCtx)
	if err != nil {
		t.Fatalf("fetching expired claims: %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("expected 0 expired claims after worker run, got %d", len(expired))
	}
}

// ─────────────────────────────────────────────────────────────
// Flow 4: Failed payment releases claim and restores inventory
// ─────────────────────────────────────────────────────────────

func TestFlow_FailedPayment_ReleasesClaimAndRestoresInventory(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	client := sharedEnv.client
	orgClient := mustLoginOrganizer(t, client, sharedEnv.pool)
	eventID := mustCreateAndActivateEvent(t, orgClient, sharedEnv.pool, sharedEnv.rdb, 10)

	rc, _ := redisstore.NewClient(testCtx, sharedEnv.rdb)
	invStore := redisstore.NewInventoryStore(rc)
	invStore.Set(testCtx, eventID, 10)

	// Customer claims
	custToken := mustCustomerAuth(t, client, sharedEnv.rdb)
	custClient := client.WithToken(custToken)

	resp := custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	if err := sharedEnv.admissionWorker.Run(testCtx); err != nil {
		t.Fatalf("admission worker: %v", err)
	}

	admToken := mustGetAdmissionToken(t, custClient, eventID)
	resp = custClient.POST(t, "/events/"+eventID+"/claims", map[string]any{
		"admission_token": admToken,
	})
	assertStatus(t, resp, http.StatusCreated)

	type claimResp struct {
		ClaimID string `json:"claim_id"`
	}
	cr := decodeBody[claimResp](t, resp)
	claimID := cr.ClaimID

	// Inventory at 9
	count, _ := invStore.Get(testCtx, eventID)
	if count != 9 {
		t.Fatalf("expected 9 after claim, got %d", count)
	}

	// Initialize payment
	ref := fmt.Sprintf("fq-fail-%d", time.Now().UnixNano())
	sharedEnv.gateway.On("InitializeTransaction", mock.Anything, mock.Anything).
		Return(&gateway.InitializeResponse{
			AuthorizationURL: "https://checkout.paystack.com/pay/test",
			Reference:        ref,
		}, nil).Once()

	resp = custClient.POST(t, "/claims/"+claimID+"/payments", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Paystack sends charge.failed
	sharedEnv.gateway.On("VerifyWebhookSignature", mock.Anything, "valid-sig").Return(true)

	failPayload := map[string]any{
		"event": "charge.failed",
		"data": map[string]any{
			"reference":        ref,
			"status":           "failed",
			"gateway_response": "Insufficient funds",
		},
	}
	req := mustBuildWebhookRequest(t, sharedEnv.client.baseURL+"/webhooks/paystack", failPayload, "valid-sig")
	webhookResp, err := sharedEnv.client.http.Do(req)
	if err != nil {
		t.Fatalf("webhook: %v", err)
	}
	assertStatus(t, webhookResp, http.StatusOK)
	webhookResp.Body.Close()

	// Inventory must be restored
	count, _ = invStore.Get(testCtx, eventID)
	if count != 10 {
		t.Fatalf("expected inventory 10 after failed payment, got %d", count)
	}

	// Claim must be RELEASED
	db := postgres.NewDB(sharedEnv.pool)
	claim, err := postgres.NewClaimStore(db).GetByID(testCtx, claimID)
	if err != nil {
		t.Fatalf("fetching claim: %v", err)
	}
	if claim.Status != domain.ClaimStatusReleased {
		t.Fatalf("expected RELEASED, got %s", claim.Status)
	}
}

// ─────────────────────────────────────────────────────────────
// Flow 5: Auth flows
// ─────────────────────────────────────────────────────────────

func TestFlow_Auth_WrongPassword(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	email, _ := seedOrganizer(testCtx, t, sharedEnv.pool)

	resp := sharedEnv.client.POST(t, "/auth/organizer/login", map[string]any{
		"email":    email,
		"password": "wrongpassword",
	})
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestFlow_Auth_WrongOTP(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	email := fmt.Sprintf("cust-%d@test.com", time.Now().UnixNano())
	resp := sharedEnv.client.POST(t, "/auth/customer/otp/request", map[string]any{
		"email": email,
	})
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	resp = sharedEnv.client.POST(t, "/auth/customer/otp/verify", map[string]any{
		"email": email,
		"otp":   "000000",
	})
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestFlow_Auth_NoTokenOnProtectedRoute(t *testing.T) {
	resp := sharedEnv.client.POST(t, "/events", map[string]any{
		"name": "test",
	})
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestFlow_Auth_CustomerTokenOnOrganizerRoute(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	custToken := mustCustomerAuth(t, sharedEnv.client, sharedEnv.rdb)
	custClient := sharedEnv.client.WithToken(custToken)

	resp := custClient.POST(t, "/events", map[string]any{
		"name":            "test",
		"total_inventory": 100,
		"price":           1000,
		"sale_start":      time.Now().Add(-time.Hour).Format(time.RFC3339),
		"sale_end":        time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	// Customer token fails organizer JWT verification → 401
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

// ─────────────────────────────────────────────────────────────
// Flow 6: Queue constraints
// ─────────────────────────────────────────────────────────────

func TestFlow_Queue_DuplicateJoinRejected(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	orgClient := mustLoginOrganizer(t, sharedEnv.client, sharedEnv.pool)
	eventID := mustCreateAndActivateEvent(t, orgClient, sharedEnv.pool, sharedEnv.rdb, 100)

	custToken := mustCustomerAuth(t, sharedEnv.client, sharedEnv.rdb)
	custClient := sharedEnv.client.WithToken(custToken)

	resp := custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusConflict)
	resp.Body.Close()
}

func TestFlow_Queue_JoinInactiveEvent(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	// Create event but don't activate it
	orgClient := mustLoginOrganizer(t, sharedEnv.client, sharedEnv.pool)
	resp := orgClient.POST(t, "/events", map[string]any{
		"name":            "Draft Event",
		"total_inventory": 100,
		"price":           1000,
		"sale_start":      time.Now().Add(-time.Hour).Format(time.RFC3339),
		"sale_end":        time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	assertStatus(t, resp, http.StatusCreated)

	type evResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	evResult := decodeBody[evResp](t, resp)
	eventID := evResult.Data.ID

	custToken := mustCustomerAuth(t, sharedEnv.client, sharedEnv.rdb)
	custClient := sharedEnv.client.WithToken(custToken)

	resp = custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestFlow_Queue_AbandonWaitingEntry(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	orgClient := mustLoginOrganizer(t, sharedEnv.client, sharedEnv.pool)
	eventID := mustCreateAndActivateEvent(t, orgClient, sharedEnv.pool, sharedEnv.rdb, 100)

	custToken := mustCustomerAuth(t, sharedEnv.client, sharedEnv.rdb)
	custClient := sharedEnv.client.WithToken(custToken)

	resp := custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = custClient.DELETE(t, "/events/"+eventID+"/queue")
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Can rejoin after abandoning
	resp = custClient.POST(t, "/events/"+eventID+"/queue", nil)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()
}

// ─────────────────────────────────────────────────────────────
// Flow 7: Organizer cannot manage another organizer's event
// ─────────────────────────────────────────────────────────────

func TestFlow_Event_OrganizerOwnership(t *testing.T) {
	truncateAll(testCtx, t, sharedEnv.pool, sharedEnv.rdb)

	org1Client := mustLoginOrganizer(t, sharedEnv.client, sharedEnv.pool)
	org2Client := mustLoginOrganizer(t, sharedEnv.client, sharedEnv.pool)

	eventID := mustCreateAndActivateEvent(t, org1Client, sharedEnv.pool, sharedEnv.rdb, 100)

	// Org2 tries to end org1's event
	resp := org2Client.PUT(t, "/events/"+eventID+"/end")
	assertStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()
}

// ─────────────────────────────────────────────────────────────
// Shared flow helpers — reduce repetition across tests
// ─────────────────────────────────────────────────────────────

func mustLoginOrganizer(t *testing.T, client *testClient, pool *pgxpool.Pool) *testClient {
	t.Helper()
	email, password := seedOrganizer(testCtx, t, pool)
	resp := client.POST(t, "/auth/organizer/login", map[string]any{
		"email":    email,
		"password": password,
	})
	assertStatus(t, resp, http.StatusOK)
	type r struct {
		Token string `json:"token"`
	}
	result := decodeBody[r](t, resp)
	return client.WithToken(result.Token)
}

func mustCustomerAuth(t *testing.T, client *testClient, rdb *redis.Client) string {
	t.Helper()
	email := fmt.Sprintf("cust-%d@test.com", time.Now().UnixNano())

	resp := client.POST(t, "/auth/customer/otp/request", map[string]any{"email": email})
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	otp, err := rdb.Get(testCtx, "otp:"+email).Result()
	if err != nil {
		t.Fatalf("reading OTP from redis: %v", err)
	}

	resp = client.POST(t, "/auth/customer/otp/verify", map[string]any{
		"email": email,
		"otp":   otp,
	})
	assertStatus(t, resp, http.StatusOK)
	type r struct {
		Token string `json:"token"`
	}
	result := decodeBody[r](t, resp)
	return result.Token
}

func mustCreateAndActivateEvent(t *testing.T, orgClient *testClient, pool *pgxpool.Pool, rdb *redis.Client, inventory int) string {
	t.Helper()

	resp := orgClient.POST(t, "/events", map[string]any{
		"name":            "Test Event",
		"total_inventory": inventory,
		"price":           10000,
		"sale_start":      time.Now().Add(-time.Hour).Format(time.RFC3339),
		"sale_end":        time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	assertStatus(t, resp, http.StatusCreated)
	type r struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	result := decodeBody[r](t, resp)
	eventID := result.Data.ID

	resp = orgClient.PUT(t, "/events/"+eventID+"/activate")
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Warm Redis inventory cache — mirrors what the startup recovery
	// worker does in production before any admission ticks run.
	rc, _ := redisstore.NewClient(testCtx, rdb)
	redisstore.NewInventoryStore(rc).Set(testCtx, eventID, int64(inventory))

	return eventID
}

func mustGetAdmissionToken(t *testing.T, custClient *testClient, eventID string) string {
	t.Helper()
	resp := custClient.GET(t, "/events/"+eventID+"/queue/position")
	assertStatus(t, resp, http.StatusOK)
	type r struct {
		Status         string  `json:"status"`
		AdmissionToken *string `json:"admission_token"`
	}
	result := decodeBody[r](t, resp)
	if result.Status != "ADMITTED" {
		t.Fatalf("expected ADMITTED status, got %s", result.Status)
	}
	if result.AdmissionToken == nil {
		t.Fatal("expected admission token, got nil")
	}
	return *result.AdmissionToken
}

func mustBuildWebhookRequest(t *testing.T, url string, body any, signature string) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshalling webhook body: %v", err)
	}
	req, err := http.NewRequestWithContext(testCtx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("building webhook request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-paystack-signature", signature)
	return req
}
