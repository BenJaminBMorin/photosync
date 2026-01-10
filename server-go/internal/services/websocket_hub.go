package services

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	ID         string
	UserID     string // Set after authentication
	Topics     map[string]bool
	Conn       *websocket.Conn
	Send       chan []byte
	hub        *WebSocketHub
	mu         sync.Mutex
	closedOnce sync.Once
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WSClient]bool
	topics     map[string]map[*WSClient]bool // topic -> clients
	userConns  map[string]map[*WSClient]bool // userID -> clients
	register   chan *WSClient
	unregister chan *WSClient
	broadcast  chan *broadcastMsg
	mu         sync.RWMutex
}

type broadcastMsg struct {
	topic   string
	userID  string // if set, only send to this user
	message []byte
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WSClient]bool),
		topics:     make(map[string]map[*WSClient]bool),
		userConns:  make(map[string]map[*WSClient]bool),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		broadcast:  make(chan *broadcastMsg, 256),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// Remove from all topics
				for topic := range client.Topics {
					if topicClients, ok := h.topics[topic]; ok {
						delete(topicClients, client)
						if len(topicClients) == 0 {
							delete(h.topics, topic)
						}
					}
				}
				// Remove from user connections
				if client.UserID != "" {
					if userClients, ok := h.userConns[client.UserID]; ok {
						delete(userClients, client)
						if len(userClients) == 0 {
							delete(h.userConns, client.UserID)
						}
					}
				}
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected: %s", client.ID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			var targets map[*WSClient]bool

			if msg.userID != "" {
				// Send to specific user
				targets = h.userConns[msg.userID]
			} else if msg.topic != "" {
				// Send to topic subscribers
				targets = h.topics[msg.topic]
			} else {
				// Broadcast to all
				targets = h.clients
			}

			for client := range targets {
				select {
				case client.Send <- msg.message:
				default:
					// Client buffer full, close connection
					go func(c *WSClient) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub
func (h *WebSocketHub) Register(client *WSClient) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *WebSocketHub) Unregister(client *WSClient) {
	h.unregister <- client
}

// Subscribe adds a client to a topic
func (h *WebSocketHub) Subscribe(client *WSClient, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.Topics[topic] = true
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*WSClient]bool)
	}
	h.topics[topic][client] = true
	log.Printf("Client %s subscribed to topic: %s", client.ID, topic)
}

// Unsubscribe removes a client from a topic
func (h *WebSocketHub) Unsubscribe(client *WSClient, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(client.Topics, topic)
	if topicClients, ok := h.topics[topic]; ok {
		delete(topicClients, client)
		if len(topicClients) == 0 {
			delete(h.topics, topic)
		}
	}
}

// SetUserID associates a client with a user
func (h *WebSocketHub) SetUserID(client *WSClient, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from old user mapping if exists
	if client.UserID != "" && client.UserID != userID {
		if userClients, ok := h.userConns[client.UserID]; ok {
			delete(userClients, client)
			if len(userClients) == 0 {
				delete(h.userConns, client.UserID)
			}
		}
	}

	client.UserID = userID
	if h.userConns[userID] == nil {
		h.userConns[userID] = make(map[*WSClient]bool)
	}
	h.userConns[userID][client] = true
}

// BroadcastToTopic sends a message to all clients subscribed to a topic
func (h *WebSocketHub) BroadcastToTopic(topic string, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	h.broadcast <- &broadcastMsg{
		topic:   topic,
		message: data,
	}
}

// SendToUser sends a message to all connections of a specific user
func (h *WebSocketHub) SendToUser(userID string, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	h.broadcast <- &broadcastMsg{
		userID:  userID,
		message: data,
	}
}

// BroadcastAll sends a message to all connected clients
func (h *WebSocketHub) BroadcastAll(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	h.broadcast <- &broadcastMsg{
		message: data,
	}
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetTopicSubscriberCount returns the number of subscribers for a topic
func (h *WebSocketHub) GetTopicSubscriberCount(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.topics[topic]; ok {
		return len(clients)
	}
	return 0
}

// NewClient creates a new WebSocket client connected to this hub
func (h *WebSocketHub) NewClient(id string, conn *websocket.Conn) *WSClient {
	return &WSClient{
		ID:     id,
		Topics: make(map[string]bool),
		Conn:   conn,
		Send:   make(chan []byte, 256),
		hub:    h,
	}
}

// WSClient methods

// Close closes the client connection
func (c *WSClient) Close() {
	c.closedOnce.Do(func() {
		c.hub.Unregister(c)
		c.Conn.Close()
	})
}

// WritePump pumps messages from the hub to the websocket connection
func (c *WSClient) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.mu.Lock()
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			c.mu.Unlock()

			if err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *WSClient) ReadPump(onMessage func(client *WSClient, messageType int, data []byte)) {
	defer c.Close()

	c.Conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if onMessage != nil {
			onMessage(c, messageType, message)
		}
	}
}

// Common message types
const (
	WSTypeAuthStatus      = "auth_status"
	WSTypeScannerProgress = "scanner_progress"
	WSTypeScannerComplete = "scanner_complete"
	WSTypeOrphanFound     = "orphan_found"
	WSTypeConflictFound   = "conflict_found"
	WSTypePhotoUploaded   = "photo_uploaded"
	WSTypeError           = "error"
	WSTypeSubscribe       = "subscribe"
	WSTypeUnsubscribe     = "unsubscribe"
	WSTypePing            = "ping"
	WSTypePong            = "pong"
)

// Common topics
const (
	TopicAuth       = "auth"
	TopicScanner    = "scanner"
	TopicAdmin      = "admin"
	TopicUserPhotos = "user_photos" // prefix with user ID: user_photos:{userID}
)

// AuthStatusPayload is sent when auth status changes
type AuthStatusPayload struct {
	RequestID    string `json:"requestId"`
	Status       string `json:"status"`
	SessionToken string `json:"sessionToken,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
}

// ScannerProgressPayload is sent during scanning
type ScannerProgressPayload struct {
	Running        bool    `json:"running"`
	FilesScanned   int     `json:"filesScanned"`
	OrphansFound   int     `json:"orphansFound"`
	ConflictsFound int     `json:"conflictsFound"`
	Progress       float64 `json:"progress"`
	CurrentFile    string  `json:"currentFile,omitempty"`
}
