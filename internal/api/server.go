// Package api implements the FairQueue HTTP API.
//
// @title           FairQueue API
// @version         1.0
// @description     Fair inventory allocation for high-demand live events in Nigeria.
// @host            localhost:8080
// @BasePath        /
//
// @securityDefinitions.apikey OrganizerAuth
// @in header
// @name Authorization
// @description JWT issued on organizer login. Format: Bearer {token}
//
// @securityDefinitions.apikey CustomerAuth
// @in header
// @name Authorization
// @description JWT issued on OTP verification. Format: Bearer {token}
package api

import (
	"net/http"

	"github.com/DanielPopoola/fairqueue/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter wires the chi router with auth middleware groups.
func NewRouter(
	h *Handlers,
	orgAuth *auth.OrganizerTokenizer,
	custAuth *auth.CustomerTokenizer,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Swagger UI — served at /swagger/index.html
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// ── Public routes ─────────────────────────────────────────
	r.Get("/health", h.HealthCheck)
	r.Post("/auth/organizer/login", h.OrganizerLogin)
	r.Post("/auth/customer/otp/request", h.RequestOTP)
	r.Post("/auth/customer/otp/verify", h.VerifyOTP)
	r.Get("/events/{eventId}", h.GetEvent)
	r.Post("/webhooks/paystack", h.PaystackWebhook)

	// ── Organizer-protected routes ────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(OrganizerAuthMiddleware(orgAuth))
		r.Post("/events", h.CreateEvent)
		r.Put("/events/{eventId}/activate", h.ActivateEvent)
		r.Put("/events/{eventId}/end", h.EndEvent)
	})

	// ── Customer-protected routes ─────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(CustomerAuthMiddleware(custAuth))
		r.Post("/events/{eventId}/queue", h.JoinQueue)
		r.Get("/events/{eventId}/queue/position", h.GetQueuePosition)
		r.Delete("/events/{eventId}/queue", h.AbandonQueue)
		r.Post("/events/{eventId}/claims", h.ClaimTicket)
		r.Delete("/claims/{claimId}", h.ReleaseClaim)
		r.Post("/claims/{claimId}/payments", h.InitializePayment)
	})

	// ── WebSocket — token auth from query param ───────────────
	r.Get("/ws/queue/{eventId}", h.QueueWebSocket)

	// Consistent JSON 404 / 405
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	})

	return r
}
