package agenthub

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/Sarin-jacob/Initiate/internal/db"
)

// HandleWS upgrades the HTTP request and performs the cryptographic handshake
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// 1. Wait for AGENT_HELLO
	var hello WSPayload
	if err := conn.ReadJSON(&hello); err != nil || hello.Event != "AGENT_HELLO" {
		log.Println("Invalid handshake initialization")
		return
	}

	pubKeyHex, _ := hello.Payload["public_key"].(string)
	if pubKeyHex == "" {
		log.Println("Missing public key in handshake")
		return
	}

	// 2. Look up Edge Agent in Database
	var target db.TargetServer
	if err := h.db.Where("public_key = ?", pubKeyHex).First(&target).Error; err != nil {
		log.Printf("Unauthorized Agent attempt. Key not registered: %s", pubKeyHex)
		return
	}

	// 3. Generate and send Cryptographic Challenge
	nonceBytes := make([]byte, 32)
	rand.Read(nonceBytes)
	nonceHex := hex.EncodeToString(nonceBytes)

	conn.WriteJSON(WSPayload{
		Event: "CHALLENGE",
		Nonce: nonceHex,
	})

	// 4. Wait for Signature Response
	var response WSPayload
	if err := conn.ReadJSON(&response); err != nil || response.Event != "CHALLENGE_RESPONSE" {
		log.Println("Agent failed to respond to challenge")
		return
	}

	// 5. Verify Ed25519 Signature
	pubKeyBytes, _ := hex.DecodeString(target.PublicKey)
	sigBytes, _ := hex.DecodeString(response.Signature)

	if len(pubKeyBytes) != ed25519.PublicKeySize || len(sigBytes) != ed25519.SignatureSize {
		log.Println("Invalid key or signature byte size")
		return
	}

	if !ed25519.Verify(pubKeyBytes, nonceBytes, sigBytes) {
		log.Printf("Cryptographic verification failed for server: %s", target.Name)
		conn.WriteJSON(WSPayload{Event: "AUTH_FAILED"})
		return
	}

	// 6. Success! Register the Client
	conn.WriteJSON(WSPayload{Event: "AUTHENTICATED"})
	
	client := &AgentClient{
		ServerID: target.ID,
		Conn:     conn,
	}
	h.registerClient(client)

	// Update DB status
	h.db.Model(&target).Updates(map[string]interface{}{
		"status":    "ONLINE",
		"last_seen": time.Now(),
	})

	// 7. Start listening for Task Results
	h.listen(client)
}

func (h *Hub) registerClient(client *AgentClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ServerID] = client
	log.Printf("Agent connected and authenticated: %s", client.ServerID)
}

func (h *Hub) unregisterClient(serverID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if client, ok := h.clients[serverID]; ok {
		client.Conn.Close()
		delete(h.clients, serverID)
		
		// Set DB status to offline
		h.db.Model(&db.TargetServer{}).Where("id = ?", serverID).Update("status", "OFFLINE")
		log.Printf("Agent disconnected: %s", serverID)
	}
}