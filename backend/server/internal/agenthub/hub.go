package agenthub

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
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
