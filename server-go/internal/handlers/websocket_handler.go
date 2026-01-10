package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/services"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - can be restricted in production
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub         *services.WebSocketHub
	authService *services.AuthService
}

// NewWebSocketHandler creates a new WebSocketHandler
func NewWebSocketHandler(hub *services.WebSocketHub, authService *services.AuthService) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		authService: authService,
	}
}

// HandleConnection upgrades HTTP to WebSocket and manages the connection
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	clientID := uuid.New().String()
	client := h.hub.NewClient(clientID, conn)

	// Check if user is authenticated via session cookie
	if session := middleware.GetSessionFromContext(r.Context()); session != nil {
		if user := middleware.GetUserFromContext(r.Context()); user != nil {
			h.hub.SetUserID(client, user.ID)
		}
	}

	h.hub.Register(client)

	// Start the write pump in a goroutine
	go client.WritePump()

	// Run the read pump (blocks until connection closes)
	client.ReadPump(h.handleMessage)
}

// HandleAuthConnection is a WebSocket endpoint specifically for auth status updates
// It subscribes to auth updates for a specific request ID
func (h *WebSocketHandler) HandleAuthConnection(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	if requestID == "" {
		http.Error(w, "requestId query parameter required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	clientID := uuid.New().String()
	client := h.hub.NewClient(clientID, conn)

	h.hub.Register(client)

	// Subscribe to auth updates for this specific request
	topic := "auth:" + requestID
	h.hub.Subscribe(client, topic)

	log.Printf("Auth WebSocket connected for request: %s", requestID)

	// Start the write pump
	go client.WritePump()

	// Run the read pump
	client.ReadPump(h.handleMessage)
}

// handleMessage processes incoming WebSocket messages
func (h *WebSocketHandler) handleMessage(client *services.WSClient, messageType int, data []byte) {
	if messageType != websocket.TextMessage {
		return
	}

	var msg services.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Invalid WebSocket message: %v", err)
		return
	}

	switch msg.Type {
	case services.WSTypeSubscribe:
		if topic, ok := msg.Payload.(string); ok {
			h.hub.Subscribe(client, topic)
		} else if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if topic, ok := payload["topic"].(string); ok {
				h.hub.Subscribe(client, topic)
			}
		}

	case services.WSTypeUnsubscribe:
		if topic, ok := msg.Payload.(string); ok {
			h.hub.Unsubscribe(client, topic)
		} else if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if topic, ok := payload["topic"].(string); ok {
				h.hub.Unsubscribe(client, topic)
			}
		}

	case services.WSTypePing:
		// Respond with pong
		response := services.WSMessage{
			Type:    services.WSTypePong,
			Payload: nil,
		}
		if data, err := json.Marshal(response); err == nil {
			client.Send <- data
		}

	default:
		log.Printf("Unknown WebSocket message type: %s", msg.Type)
	}
}

// NotifyAuthStatus sends auth status update to clients waiting for a specific request
func (h *WebSocketHandler) NotifyAuthStatus(requestID, status, sessionToken string) {
	topic := "auth:" + requestID
	payload := services.AuthStatusPayload{
		RequestID:    requestID,
		Status:       status,
		SessionToken: sessionToken,
	}

	h.hub.BroadcastToTopic(topic, services.WSMessage{
		Type:    services.WSTypeAuthStatus,
		Payload: payload,
	})

	log.Printf("Sent auth status notification for request %s: %s", requestID, status)
}

// GetHub returns the WebSocket hub (for other services to send notifications)
func (h *WebSocketHandler) GetHub() *services.WebSocketHub {
	return h.hub
}
