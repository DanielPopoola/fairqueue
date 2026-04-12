package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/DanielPopoola/fairqueue/internal/service"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
)

// Handlers holds all dependencies for HTTP handlers.
// Each handler is a method on this struct — one job: decode input,
// call a service, encode output. No business logic lives here.
type Handlers struct {
	events     *service.EventService
	claimSvc   *service.ClaimService
	queueSvc   *service.QueueService
	paymentSvc *service.PaymentService
	customers  *postgres.CustomerStore
	organizers *postgres.OrganizerStore
	claims     *postgres.ClaimStore
	orgTokens  *auth.OrganizerTokenizer
	custTokens *auth.CustomerTokenizer
	otpStore   *OTPStore
	hub        *Hub
	db         interface{ Ping(context.Context) error }
	rdb        interface{ Ping(context.Context) error }
	logger     *slog.Logger
}

func NewHandlers(
	events *service.EventService,
	claimSvc *service.ClaimService,
	queueSvc *service.QueueService,
	paymentSvc *service.PaymentService,
	customers *postgres.CustomerStore,
	organizers *postgres.OrganizerStore,
	claims *postgres.ClaimStore,
	orgTokens *auth.OrganizerTokenizer,
	custTokens *auth.CustomerTokenizer,
	otpStore *OTPStore,
	hub *Hub,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		events:     events,
		claimSvc:   claimSvc,
		queueSvc:   queueSvc,
		paymentSvc: paymentSvc,
		customers:  customers,
		organizers: organizers,
		claims:     claims,
		orgTokens:  orgTokens,
		custTokens: custTokens,
		otpStore:   otpStore,
		hub:        hub,
		logger:     logger,
	}
}

// ── Health ────────────────────────────────────────────────────

// HealthCheck godoc
// @Summary     Health check
// @Description Returns Postgres and Redis connectivity status
// @Tags        System
// @Produce     json
// @Success     200  {object}  HealthResponse
// @Failure     503  {object}  HealthResponse
// @Router      /health [get]
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	pgOK := true
	redisOK := true

	if h.db != nil {
		if err := h.db.Ping(r.Context()); err != nil {
			pgOK = false
		}
	}
	if h.rdb != nil {
		if err := h.rdb.Ping(r.Context()); err != nil {
			redisOK = false
		}
	}

	status := "healthy"
	code := http.StatusOK
	if !pgOK || !redisOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	pgStatus := "ok"
	if !pgOK {
		pgStatus = "error"
	}
	redisStatus := "ok"
	if !redisOK {
		redisStatus = "error"
	}

	writeJSON(w, code, HealthResponse{
		Status:   status,
		Postgres: pgStatus,
		Redis:    redisStatus,
	})
}

// ── Auth ─────────────────────────────────────────────────────

// OrganizerLogin godoc
// @Summary     Organizer login
// @Description Authenticates an organizer and returns a JWT
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body  body      OrganizerLoginRequest  true  "Credentials"
// @Success     200   {object}  AuthResponse
// @Failure     401   {object}  ErrorResponse
// @Failure     500   {object}  ErrorResponse
// @Router      /auth/organizer/login [post]
func (h *Handlers) OrganizerLogin(w http.ResponseWriter, r *http.Request) {
	var req OrganizerLoginRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	organizer, err := h.organizers.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect")
			return
		}
		writeInternalError(w, h.logger, err)
		return
	}

	if err := auth.CheckPassword(organizer.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect")
		return
	}

	token, err := h.orgTokens.Generate(organizer.ID)
	if err != nil {
		writeInternalError(w, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		Success:     true,
		Token:       token,
		OrganizerID: organizer.ID,
	})
}

// RequestOTP godoc
// @Summary     Request OTP
// @Description Sends a 6-digit OTP to the customer's email. Creates account if needed.
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body  body      OTPRequestBody  true  "Email"
// @Success     200   {object}  MessageResponse
// @Failure     500   {object}  ErrorResponse
// @Router      /auth/customer/otp/request [post]
func (h *Handlers) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req OTPRequestBody
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	if _, err := h.customers.GetOrCreate(r.Context(), req.Email); err != nil {
		writeInternalError(w, h.logger, err)
		return
	}

	code, err := generateOTP()
	if err != nil {
		writeInternalError(w, h.logger, err)
		return
	}

	if err := h.otpStore.Save(r.Context(), req.Email, code); err != nil {
		writeInternalError(w, h.logger, err)
		return
	}

	// TODO(Phase 5): send via email provider
	h.logger.Info("OTP generated", "email", req.Email, "otp", code)

	writeJSON(w, http.StatusOK, MessageResponse{
		Success: true,
		Message: "OTP sent to your email",
	})
}

// VerifyOTP godoc
// @Summary     Verify OTP
// @Description Validates the OTP and returns a customer JWT
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body  body      OTPVerifyRequest  true  "Email and OTP"
// @Success     200   {object}  CustomerAuthResponse
// @Failure     401   {object}  ErrorResponse
// @Failure     500   {object}  ErrorResponse
// @Router      /auth/customer/otp/verify [post]
func (h *Handlers) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req OTPVerifyRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	if err := h.otpStore.Verify(r.Context(), req.Email, req.OTP); err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_OTP", "OTP is invalid or has expired")
		return
	}

	customer, err := h.customers.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeInternalError(w, h.logger, err)
		return
	}

	token, err := h.custTokens.Generate(customer.ID)
	if err != nil {
		writeInternalError(w, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, CustomerAuthResponse{
		Success:    true,
		Token:      token,
		CustomerID: customer.ID,
	})
}

// ── Events ───────────────────────────────────────────────────

// CreateEvent godoc
// @Summary     Create event
// @Description Creates a new event in DRAFT status
// @Tags        Events
// @Security    OrganizerAuth
// @Accept      json
// @Produce     json
// @Param       body  body      CreateEventRequest  true  "Event details"
// @Success     201   {object}  EventResponse
// @Failure     400   {object}  ErrorResponse
// @Failure     401   {object}  ErrorResponse
// @Failure     500   {object}  ErrorResponse
// @Router      /events [post]
func (h *Handlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := organizerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req CreateEventRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	event, err := h.events.Create(r.Context(), organizerID, &service.CreateEventRequest{
		Name:           req.Name,
		TotalInventory: req.TotalInventory,
		PriceNGN:       req.Price,
		SaleStart:      req.SaleStart,
		SaleEnd:        req.SaleEnd,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
		writeInternalError(w, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, EventResponse{
		Success: true,
		Data:    domainEventToResponse(event),
	})
}

// GetEvent godoc
// @Summary     Get event
// @Description Returns event details by ID
// @Tags        Events
// @Produce     json
// @Param       eventId  path      string  true  "Event ID (UUID)"
// @Success     200      {object}  EventResponse
// @Failure     404      {object}  ErrorResponse
// @Failure     500      {object}  ErrorResponse
// @Router      /events/{eventId} [get]
func (h *Handlers) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	if _, err := uuid.Parse(eventID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "eventId must be a valid UUID")
		return
	}
	event, err := h.events.Get(r.Context(), chi.URLParam(r, "eventId"))
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "event not found")
			return
		}
		writeInternalError(w, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, EventResponse{
		Success: true,
		Data:    domainEventToResponse(event),
	})
}

// ActivateEvent godoc
// @Summary     Activate event
// @Description Transitions event from DRAFT to ACTIVE
// @Tags        Events
// @Security    OrganizerAuth
// @Produce     json
// @Param       eventId  path      string  true  "Event ID (UUID)"
// @Success     200      {object}  EventResponse
// @Failure     400      {object}  ErrorResponse
// @Failure     401      {object}  ErrorResponse
// @Failure     403      {object}  ErrorResponse
// @Failure     404      {object}  ErrorResponse
// @Failure     500      {object}  ErrorResponse
// @Router      /events/{eventId}/activate [put]
func (h *Handlers) ActivateEvent(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := organizerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if _, err := uuid.Parse(eventID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "eventId must be a valid UUID")
		return
	}

	event, err := h.events.Activate(r.Context(), organizerID, chi.URLParam(r, "eventId"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEventNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "event not found")
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "you do not own this event")
		case errors.Is(err, domain.ErrInvalidTransition):
			writeError(w, http.StatusBadRequest, "INVALID_TRANSITION", "event must be in DRAFT state to activate")
		default:
			writeInternalError(w, h.logger, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, EventResponse{
		Success: true,
		Data:    domainEventToResponse(event),
	})
}

// EndEvent godoc
// @Summary     End event
// @Description Transitions event from ACTIVE or SOLD_OUT to ENDED
// @Tags        Events
// @Security    OrganizerAuth
// @Produce     json
// @Param       eventId  path      string  true  "Event ID (UUID)"
// @Success     200      {object}  EventResponse
// @Failure     400      {object}  ErrorResponse
// @Failure     401      {object}  ErrorResponse
// @Failure     403      {object}  ErrorResponse
// @Failure     404      {object}  ErrorResponse
// @Failure     500      {object}  ErrorResponse
// @Router      /events/{eventId}/end [put]
func (h *Handlers) EndEvent(w http.ResponseWriter, r *http.Request) {
	organizerID, ok := organizerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if _, err := uuid.Parse(eventID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "eventId must be a valid UUID")
		return
	}

	event, err := h.events.End(r.Context(), organizerID, chi.URLParam(r, "eventId"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEventNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "event not found")
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "you do not own this event")
		case errors.Is(err, domain.ErrInvalidTransition):
			writeError(w, http.StatusBadRequest, "INVALID_TRANSITION", "event must be ACTIVE or SOLD_OUT to end")
		default:
			writeInternalError(w, h.logger, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, EventResponse{
		Success: true,
		Data:    domainEventToResponse(event),
	})
}

// ── Queue ────────────────────────────────────────────────────

// JoinQueue godoc
// @Summary     Join queue
// @Description Adds the authenticated customer to the waiting queue
// @Tags        Queue
// @Security    CustomerAuth
// @Produce     json
// @Param       eventId  path      string  true  "Event ID (UUID)"
// @Success     201      {object}  QueueJoinResponse
// @Failure     401      {object}  ErrorResponse
// @Failure     404      {object}  ErrorResponse
// @Failure     409      {object}  ErrorResponse
// @Failure     500      {object}  ErrorResponse
// @Router      /events/{eventId}/queue [post]
func (h *Handlers) JoinQueue(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if _, err := uuid.Parse(eventID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "eventId must be a valid UUID")
		return
	}

	result, err := h.queueSvc.Join(r.Context(), customerID, chi.URLParam(r, "eventId"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEventNotFound), errors.Is(err, domain.ErrEventNotActive):
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		case errors.Is(err, domain.ErrAlreadyInQueue):
			writeError(w, http.StatusConflict, "ALREADY_IN_QUEUE", "you are already in the queue for this event")
		default:
			writeInternalError(w, h.logger, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, QueueJoinResponse{
		Success:      true,
		QueueEntryID: result.QueueEntry.ID,
		EventID:      result.QueueEntry.EventID,
		Position:     result.Position,
	})
}

// GetQueuePosition godoc
// @Summary     Get queue position
// @Description Returns the customer's current position. Returns admission token when admitted.
// @Tags        Queue
// @Security    CustomerAuth
// @Produce     json
// @Param       eventId  path      string  true  "Event ID (UUID)"
// @Success     200      {object}  QueuePositionResponse
// @Failure     401      {object}  ErrorResponse
// @Failure     404      {object}  ErrorResponse
// @Failure     500      {object}  ErrorResponse
// @Router      /events/{eventId}/queue/position [get]
func (h *Handlers) GetQueuePosition(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if _, err := uuid.Parse(eventID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "eventId must be a valid UUID")
		return
	}

	position, err := h.queueSvc.GetPosition(r.Context(), customerID, chi.URLParam(r, "eventId"))
	if err != nil {
		if errors.Is(err, domain.ErrQueueEntryNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "you are not in the queue for this event")
			return
		}
		writeInternalError(w, h.logger, err)
		return
	}

	resp := QueuePositionResponse{
		Success:  true,
		Position: position,
		Status:   "WAITING",
	}

	// position == 0 means admitted. Generate token on demand as fallback
	// for customers who missed the WebSocket push or reconnected.
	if position == 0 {
		resp.Status = "ADMITTED"
		// Only generate token if still within admission window in Postgres
		entry, err := h.queueSvc.GetAdmittedEntry(r.Context(), customerID, chi.URLParam(r, "eventId"))
		if err == nil && !entry.IsExpired() {
			token, err := h.custTokens.GenerateAdmission(customerID, chi.URLParam(r, "eventId"))
			if err != nil {
				h.logger.Warn("failed to generate admission token on position poll",
					"customer_id", customerID, "error", err,
				)
			} else {
				resp.AdmissionToken = &token
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// AbandonQueue godoc
// @Summary     Abandon queue
// @Description Removes the customer from the waiting queue. Only WAITING entries can be abandoned.
// @Tags        Queue
// @Security    CustomerAuth
// @Produce     json
// @Param       eventId  path  string  true  "Event ID (UUID)"
// @Success     204
// @Failure     400  {object}  ErrorResponse
// @Failure     401  {object}  ErrorResponse
// @Failure     404  {object}  ErrorResponse
// @Failure     500  {object}  ErrorResponse
// @Router      /events/{eventId}/queue [delete]
func (h *Handlers) AbandonQueue(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if _, err := uuid.Parse(eventID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "eventId must be a valid UUID")
		return
	}

	if err := h.queueSvc.Abandon(r.Context(), customerID, chi.URLParam(r, "eventId")); err != nil {
		switch {
		case errors.Is(err, domain.ErrQueueEntryNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "queue entry not found")
		case errors.Is(err, domain.ErrInvalidTransition):
			writeError(w, http.StatusBadRequest, "INVALID_TRANSITION", "admitted customers cannot abandon the queue")
		default:
			writeInternalError(w, h.logger, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Claims ───────────────────────────────────────────────────

// ClaimTicket godoc
// @Summary     Claim a ticket
// @Description Claims a ticket using a valid admission token. Customer has 10 minutes to pay.
// @Tags        Claims
// @Security    CustomerAuth
// @Accept      json
// @Produce     json
// @Param       eventId  path      string        true  "Event ID (UUID)"
// @Param       body     body      ClaimRequest  true  "Admission token"
// @Success     201      {object}  ClaimResponse
// @Failure     400      {object}  ErrorResponse
// @Failure     401      {object}  ErrorResponse
// @Failure     409      {object}  ErrorResponse
// @Failure     410      {object}  ErrorResponse
// @Failure     500      {object}  ErrorResponse
// @Router      /events/{eventId}/claims [post]
func (h *Handlers) ClaimTicket(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if _, err := uuid.Parse(eventID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "eventId must be a valid UUID")
		return
	}

	var req ClaimRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	// Verify token belongs to the authenticated customer before passing to service
	tokenCustomerID, _, err := h.claimSvc.VerifyAdmissionToken(req.AdmissionToken)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ADMISSION_TOKEN_EXPIRED", err.Error())
		return
	}
	if tokenCustomerID != customerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "admission token does not belong to you")
		return
	}

	result, err := h.claimSvc.Claim(r.Context(), req.AdmissionToken, chi.URLParam(r, "eventId"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyClaimed):
			writeError(w, http.StatusConflict, "ALREADY_CLAIMED", err.Error())
		case errors.Is(err, domain.ErrEventSoldOut):
			writeError(w, http.StatusGone, "EVENT_SOLD_OUT", "this event is sold out")
		default:
			writeError(w, http.StatusBadRequest, "ADMISSION_TOKEN_EXPIRED", err.Error())
		}
		return
	}

	expiresAt := result.Claim.CreatedAt.Add(domain.ClaimTTL)
	writeJSON(w, http.StatusCreated, ClaimResponse{
		Success:   true,
		ClaimID:   result.Claim.ID,
		EventID:   result.EventID,
		ExpiresAt: expiresAt,
	})
}

// ReleaseClaim godoc
// @Summary     Release a claim
// @Description Explicitly releases a claim before the TTL expires. Restores inventory.
// @Tags        Claims
// @Security    CustomerAuth
// @Produce     json
// @Param       claimId  path  string  true  "Claim ID (UUID)"
// @Success     204
// @Failure     401  {object}  ErrorResponse
// @Failure     403  {object}  ErrorResponse
// @Failure     404  {object}  ErrorResponse
// @Failure     409  {object}  ErrorResponse
// @Failure     500  {object}  ErrorResponse
// @Router      /claims/{claimId} [delete]
func (h *Handlers) ReleaseClaim(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	claimID := chi.URLParam(r, "claimId")
	if _, err := uuid.Parse(claimID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "claimId must be a valid UUID")
		return
	}

	claim, err := h.claims.GetByID(r.Context(), claimID)
	if err != nil {
		if errors.Is(err, domain.ErrClaimNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "claim not found")
			return
		}
		writeInternalError(w, h.logger, err)
		return
	}
	if claim.CustomerID != customerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "you do not own this claim")
		return
	}

	if err := h.claimSvc.Release(r.Context(), claimID); err != nil {
		if errors.Is(err, domain.ErrClaimNotClaimable) {
			writeError(w, http.StatusConflict, "INVALID_TRANSITION", "claim cannot be released in its current state")
			return
		}
		writeInternalError(w, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Payments ─────────────────────────────────────────────────

// InitializePayment godoc
// @Summary     Initialize payment
// @Description Creates a Paystack transaction for a claim. Idempotent — safe to call multiple times.
// @Tags        Payments
// @Security    CustomerAuth
// @Produce     json
// @Param       claimId  path      string  true  "Claim ID (UUID)"
// @Success     201      {object}  PaymentInitResponse
// @Failure     400      {object}  ErrorResponse
// @Failure     401      {object}  ErrorResponse
// @Failure     403      {object}  ErrorResponse
// @Failure     404      {object}  ErrorResponse
// @Failure     500      {object}  ErrorResponse
// @Router      /claims/{claimId}/payments [post]
func (h *Handlers) InitializePayment(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	claimID := chi.URLParam(r, "claimId")
	if _, err := uuid.Parse(claimID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "claimId must be a valid UUID")
		return
	}

	claim, err := h.claims.GetByID(r.Context(), claimID)
	if err != nil {
		if errors.Is(err, domain.ErrClaimNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "claim not found")
			return
		}
		writeInternalError(w, h.logger, err)
		return
	}
	if claim.CustomerID != customerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "you do not own this claim")
		return
	}

	result, err := h.paymentSvc.Initialize(r.Context(), claimID, customerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, PaymentInitResponse{
		Success:          true,
		PaymentID:        result.Payment.ID,
		AuthorizationURL: result.AuthorizationURL,
		Reference:        derefString(result.Payment.Reference),
	})
}

// PaystackWebhook godoc
// @Summary     Paystack webhook
// @Description Receives charge.success and charge.failed events from Paystack
// @Tags        Payments
// @Accept      json
// @Produce     json
// @Param       x-paystack-signature  header    string  true  "HMAC-SHA512 signature"
// @Success     200
// @Failure     400  {object}  ErrorResponse
// @Failure     500  {object}  ErrorResponse
// @Router      /webhooks/paystack [post]
func (h *Handlers) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	// Read raw body before any decoding — HMAC verification requires
	// the exact bytes Paystack sent.
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "failed to read request body")
		return
	}

	if err := h.paymentSvc.HandleWebhook(r.Context(), rawBody, r.Header.Get("x-paystack-signature")); err != nil {
		h.logger.Warn("webhook handling failed", "error", err)
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ── WebSocket ────────────────────────────────────────────────

// QueueWebSocket godoc
// @Summary     Queue position WebSocket
// @Description Upgrade to WebSocket for live queue position updates. Pass JWT as ?token= query param.
// @Tags        Queue
// @Param       eventId  path   string  true  "Event ID (UUID)"
// @Param       token    query  string  true  "Customer JWT"
// @Success     101
// @Failure     401  {object}  ErrorResponse
// @Router      /ws/queue/{eventId} [get]
func (h *Handlers) QueueWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "token query parameter required")
		return
	}

	customerID, err := h.custTokens.Verify(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}

	h.hub.ServeWS(r.Context(), w, r, customerID, h.logger)
}

// ── Request / response types ──────────────────────────────────

// OrganizerLoginRequest defines model for OrganizerLogin.
type OrganizerLoginRequest struct {
	Email    string `json:"email"    example:"organizer@example.com"`
	Password string `json:"password" example:"supersecret"`
}

// OTPRequestBody defines model for RequestOTP.
type OTPRequestBody struct {
	Email string `json:"email" example:"customer@example.com"`
}

// OTPVerifyRequest defines model for VerifyOTP.
type OTPVerifyRequest struct {
	Email string `json:"email" example:"customer@example.com"`
	OTP   string `json:"otp"   example:"482910"`
}

// CreateEventRequest defines model for CreateEvent.
type CreateEventRequest struct {
	Name           string    `json:"name"            example:"Burna Boy Live in Lagos"`
	TotalInventory int       `json:"total_inventory" example:"5000"`
	Price          int64     `json:"price"           example:"25000"`
	SaleStart      time.Time `json:"sale_start"`
	SaleEnd        time.Time `json:"sale_end"`
}

// ClaimRequest defines model for ClaimTicket.
type ClaimRequest struct {
	AdmissionToken string `json:"admission_token"`
}

// AuthResponse is the response for organizer login.
type AuthResponse struct {
	Success     bool   `json:"success"      example:"true"`
	Token       string `json:"token"`
	OrganizerID string `json:"organizer_id"`
}

// CustomerAuthResponse is the response for OTP verification.
type CustomerAuthResponse struct {
	Success    bool   `json:"success"     example:"true"`
	Token      string `json:"token"`
	CustomerID string `json:"customer_id"`
}

// MessageResponse is a generic success message.
type MessageResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"OTP sent to your email"`
}

// EventData is the event payload returned in responses.
type EventData struct {
	ID             string    `json:"id"`
	OrganizerID    string    `json:"organizer_id"`
	Name           string    `json:"name"`
	TotalInventory int       `json:"total_inventory"`
	Price          int64     `json:"price"`
	Status         string    `json:"status"`
	SaleStart      time.Time `json:"sale_start"`
	SaleEnd        time.Time `json:"sale_end"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// EventResponse wraps an event for API responses.
type EventResponse struct {
	Success bool       `json:"success" example:"true"`
	Data    *EventData `json:"data"`
}

// QueueJoinResponse is the response for joining the queue.
type QueueJoinResponse struct {
	Success      bool   `json:"success"        example:"true"`
	QueueEntryID string `json:"queue_entry_id"`
	EventID      string `json:"event_id"`
	Position     int64  `json:"position"       example:"1547"`
}

// QueuePositionResponse is the response for getting queue position.
type QueuePositionResponse struct {
	Success        bool    `json:"success"                   example:"true"`
	Position       int64   `json:"position"                  example:"847"`
	Status         string  `json:"status"                    example:"WAITING"`
	AdmissionToken *string `json:"admission_token,omitempty"`
}

// ClaimResponse is the response for claiming a ticket.
type ClaimResponse struct {
	Success   bool      `json:"success"    example:"true"`
	ClaimID   string    `json:"claim_id"`
	EventID   string    `json:"event_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PaymentInitResponse is the response for initializing payment.
type PaymentInitResponse struct {
	Success          bool   `json:"success"           example:"true"`
	PaymentID        string `json:"payment_id"`
	AuthorizationURL string `json:"authorization_url"`
	Reference        string `json:"reference"`
}

// HealthResponse is the response for the health check.
type HealthResponse struct {
	Status   string `json:"status"   example:"healthy"`
	Postgres string `json:"postgres" example:"ok"`
	Redis    string `json:"redis"    example:"ok"`
}

// ErrorResponse is returned on all errors.
type ErrorResponse struct {
	Success bool        `json:"success" example:"false"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail contains the machine-readable code and human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"    example:"NOT_FOUND"`
	Message string `json:"message" example:"event not found"`
}

// ── Helpers ───────────────────────────────────────────────────

type contextKey string

const (
	organizerIDKey contextKey = "organizer_id"
	customerIDKey  contextKey = "customer_id"
)

func organizerIDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(organizerIDKey).(string)
	return id, ok && id != ""
}

func customerIDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(customerIDKey).(string)
	return id, ok && id != ""
}

const koboPerNaira int64 = 100

func domainEventToResponse(e *domain.Event) *EventData {
	return &EventData{
		ID:             e.ID,
		OrganizerID:    e.OrganizerID,
		Name:           e.Name,
		TotalInventory: e.TotalInventory,
		Price:          e.Price / koboPerNaira,
		Status:         string(e.Status),
		SaleStart:      e.SaleStart,
		SaleEnd:        e.SaleEnd,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck // error check not necessary here
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Success: false,
		Error:   ErrorDetail{Code: code, Message: message},
	})
}

func writeInternalError(w http.ResponseWriter, logger *slog.Logger, err error) {
	logger.Error("internal error", "error", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}

func decode(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *Handlers) WithHealthDeps(db, rdb interface{ Ping(context.Context) error }) *Handlers {
	h.db = db
	h.rdb = rdb
	return h
}
