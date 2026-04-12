package api

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// WSMessage is the envelope pushed to connected customers.
//
//	{ "type": "position",  "position": 847 }
//	{ "type": "admitted",  "admission_token": "eyJ..." }
//	{ "type": "expired" }
type WSMessage struct {
	Type           string `json:"type"`
	Position       int64  `json:"position,omitempty"`
	AdmissionToken string `json:"admission_token,omitempty"`
}

// Hub maintains active WebSocket connections keyed by customer ID.
// The admission worker calls NotifyAdmitted to push tokens to waiting customers.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string]*websocket.Conn)}
}

// NotifyAdmitted pushes an admission token to the customer if they are
// currently connected. Returns false if the customer has no active connection.
func (h *Hub) NotifyAdmitted(ctx context.Context, customerID, token string, logger *slog.Logger) bool {
	sent := h.send(ctx, customerID, WSMessage{
		Type:           "admitted",
		AdmissionToken: token,
	})
	if !sent {
		logger.Info("customer not connected via WebSocket — token retrievable via poll",
			"customer_id", customerID,
		)
	}
	return sent
}

// BroadcastPosition pushes a position update to a connected customer.
func (h *Hub) BroadcastPosition(ctx context.Context, customerID string, position int64) bool {
	return h.send(ctx, customerID, WSMessage{
		Type:     "position",
		Position: position,
	})
}

func (h *Hub) send(ctx context.Context, customerID string, msg WSMessage) bool {
	h.mu.RLock()
	conn, ok := h.conns[customerID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	if err := wsjson.Write(ctx, conn, msg); err != nil {
		h.remove(customerID)
		return false
	}
	return true
}

func (h *Hub) register(customerID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[customerID] = conn
}

func (h *Hub) remove(customerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, customerID)
}

// ServeWS upgrades the connection, registers it, and blocks until it closes.
func (h *Hub) ServeWS(ctx context.Context, w http.ResponseWriter, r *http.Request, customerID string, logger *slog.Logger) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Phase 5: restrict to known frontend origins
		InsecureSkipVerify: true,
	})
	if err != nil {
		logger.Warn("websocket upgrade failed", "customer_id", customerID, "error", err)
		return
	}
	defer conn.CloseNow() //nolint:errcheck // error check not necessary here

	h.register(customerID, conn)
	defer h.remove(customerID)

	logger.Info("websocket connected", "customer_id", customerID)

	// Server-push only — read loop detects disconnects
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			logger.Info("websocket disconnected", "customer_id", customerID)
			return
		}
	}
}
