package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"

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
	
	// Inject the Public Key into the HTTP upgrade request headers
	headers := http.Header{}
	headers.Add("X-Agent-Pubkey", pubKey)
	
	// Pass the headers into the dialer instead of 'nil'
	conn, resp, err := dialer.Dial(config.Server.URL, headers)
	if err != nil {
		if resp != nil {
			log.Fatalf("Failed to connect: %v (Status: %s)", err, resp.Status)
		}
		log.Fatalf("Failed to connect to Central Server at %s: %v", config.Server.URL, err)
	}
	defer conn.Close()
	log.Println("Connected. Initiating Handshake...")

	// 5. Authenticate
	hello := WSPayload{Event: "AGENT_HELLO", Payload: map[string]interface{}{"public_key": pubKey}}
	conn.WriteJSON(hello)

	var challenge WSPayload
	conn.ReadJSON(&challenge)
	sig := internal.SignChallenge(privKey, challenge.Nonce)
	conn.WriteJSON(WSPayload{Event: "CHALLENGE_RESPONSE", Signature: sig})

	var authConfirm WSPayload
	conn.ReadJSON(&authConfirm)
	if authConfirm.Event != "AUTHENTICATED" {
		log.Fatalf("Authentication failed. Server responded: %s", authConfirm.Event)
	}
	log.Println("Successfully Authenticated. Listening for tasks...")

	// 6. Execution Loop
	for {
		var msg WSPayload
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Connection closed or error:", err)
			break
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
}