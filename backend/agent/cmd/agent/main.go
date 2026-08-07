package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"

	"github.com/Sarin-jacob/Initiate-Agent/internal"
)

type WSPayload struct {
	Event     string                 `json:"event"`
	TaskID    string                 `json:"task_id,omitempty"`
	Signature string                 `json:"signature,omitempty"`
	Nonce     string                 `json:"nonce,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

func main() {
	// 1. Parse CLI Flags
	configPath := flag.String("config", "config.yaml", "Path to config.yaml")
	flag.Parse()

	// 2. Read and Parse config.yaml
	configData, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file at %s: %v", *configPath, err)
	}

	var config internal.AgentConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}

	// 3. Load Cryptographic Keys
	privKey, err := internal.LoadPrivateKey(config.Server.PrivateKeyPath)
	if err != nil {
		log.Fatalf("Failed to load private key from %s: %v", config.Server.PrivateKeyPath, err)
	}
	pubKey := hex.EncodeToString(privKey.Public().(ed25519.PublicKey))

	// 4. Dial WebSocket with Pinned TLS
	dialer := websocket.Dialer{
		TLSClientConfig: internal.GetPinnedTLSConfig(config.Server.CertPin),
	}
	
	headers := http.Header{}
	headers.Add("X-Agent-Pubkey", pubKey)

	// --- INFINITE RECONNECT LOOP ---
	for {
		log.Printf("Attempting to connect to %s...", config.Server.URL)
		conn, resp, err := dialer.Dial(config.Server.URL, headers)
		if err != nil {
			status := "unknown"
			if resp != nil {
				status = resp.Status
			}
			log.Printf("Connection failed: %v (Status: %s). Retrying in 5 seconds...", err, status)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Connected. Initiating Handshake...")

		// 5. Authenticate
		conn.WriteJSON(WSPayload{Event: "AGENT_HELLO", Payload: map[string]interface{}{"public_key": pubKey}})

		var challenge WSPayload
		conn.ReadJSON(&challenge)
		sig := internal.SignChallenge(privKey, challenge.Nonce)
		conn.WriteJSON(WSPayload{Event: "CHALLENGE_RESPONSE", Signature: sig})

		var authConfirm WSPayload
		conn.ReadJSON(&authConfirm)
		if authConfirm.Event != "AUTHENTICATED" {
			log.Printf("Authentication failed. Server responded: %s", authConfirm.Event)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		
		log.Println("Successfully Authenticated. Listening for tasks...")

		// 6. Execution Loop
		for {
			var msg WSPayload
			if err := conn.ReadJSON(&msg); err != nil {
				log.Println("Connection dropped or error:", err)
				break // Break inner loop to trigger reconnect
			}

			log.Printf("Received task: %s (ID: %s)", msg.Event, msg.TaskID)

			cmdConf, exists := config.Executor[msg.Event]
			if !exists {
				log.Printf("Warning: No executor configured for event %s", msg.Event)
				continue
			}

			outStr, execErr := internal.ExecuteTask(cmdConf, msg.Payload)
			
			status := "SUCCESS"
			errorMsg := ""
			if execErr != nil {
				status = "FAILED"
				errorMsg = execErr.Error()
			}

			conn.WriteJSON(WSPayload{
				Event:  "TASK_RESULT",
				TaskID: msg.TaskID,
				Payload: map[string]interface{}{
					"status": status,
					"output": outStr,
					"error":  errorMsg,
				},
			})
			log.Printf("Task %s completed with status %s", msg.TaskID, status)
		}

		// If we break out of the execution loop, clean up and wait before reconnecting
		conn.Close()
		log.Println("Disconnected from Central Server. Retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}