package agenthub

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

// WSPayload must exactly match the Edge Agent's expected JSON structure
type WSPayload struct {
	Event     string                 `json:"event"`
	TaskID    string                 `json:"task_id,omitempty"`
	Signature string                 `json:"signature,omitempty"`
	Nonce     string                 `json:"nonce,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// AgentClient represents an actively connected and authenticated Edge Agent
type AgentClient struct {
	ServerID string
	Conn     *websocket.Conn
}

// Hub manages all active WebSocket connections to Edge Agents
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*AgentClient
	db      *gorm.DB
}

// NewHub initializes the WebSocket hub
func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		clients: make(map[string]*AgentClient),
		db:      db,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Ensure we only allow connections from our expected origin (or allow all if behind Nginx)
	CheckOrigin: func(r *http.Request) bool { return true }, 
}


// Dispatch sends a targeted event payload to a specific Edge Agent by its ID
func (h *Hub) Dispatch(serverID string, event string, payload map[string]interface{}) {
	h.mu.RLock()
	client, exists := h.clients[serverID]
	h.mu.RUnlock()

	if !exists {
		// If the server is offline, you could implement a queue system here later
		// For now, we just log that the server wasn't reachable immediately.
		h.logger.Printf("Warning: Attempted to dispatch '%s' to %s, but agent is offline.", event, serverID)
		return
	}

	taskID := uuid.New().String()
	
	msg := WSPayload{
		Event:   event,
		TaskID:  taskID,
		Payload: payload,
	}

	// Send the JSON payload over the WebSocket connection
	if err := client.Conn.WriteJSON(msg); err != nil {
		h.logger.Printf("Failed to dispatch task %s to server %s: %v", taskID, serverID, err)
		
		// If the write fails, the connection is likely dead, so we remove the client
		h.removeClient(client)
	} else {
		h.logger.Printf("Successfully dispatched task '%s' (ID: %s) to server %s", event, taskID, serverID)
	}
}