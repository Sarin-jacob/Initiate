package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Sarin-jacob/Initiate-Agent/internal"
)

type WSPayload struct {
	Event     string                 `json:"event"`
	TaskID    string                 `json:"task_id,omitempty"`
	Signature string                 `json:"signature,omitempty"`
	Nonce     string                 `json:"nonce,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// EnsureKeyExists creates a new Ed25519 key if one is not found at the given path
func EnsureKeyExists(path string) (ed25519.PrivateKey, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println("No private key found. Generating new Ed25519 keypair...")

		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key: %w", err)
		}

		// Save with strict 0600 permissions
		if err := os.WriteFile(path, priv, 0600); err != nil {
			return nil, fmt.Errorf("failed to save private key: %w", err)
		}

		log.Printf("New key generated successfully. Public Key: %s\n", hex.EncodeToString(pub))
		return priv, nil
	}

	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing key: %w", err)
	}

	return ed25519.PrivateKey(keyBytes), nil
}

func main() {
	// 1. Parse CLI Flags
	configPath := flag.String("config", "config.yml", "Path to config.yml")
	showPubKey := flag.Bool("show-pubkey", false, "Print the public key and exit")
	flag.Parse()

	// 2. Read Configuration
	config, err := internal.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to parse config.yml: %v", err)
	}

	// 3. Ensure Cryptographic Identity
	privKey, err := EnsureKeyExists(config.Server.PrivateKeyPath)
	if err != nil {
		log.Fatalf("Failed to load or generate identity: %v", err)
	}
	pubKey := hex.EncodeToString(privKey.Public().(ed25519.PublicKey))

	// 4. CLI Intercept: Display key and exit
	if *showPubKey {
		fmt.Printf("\n--- Edge Agent Identity ---\n")
		fmt.Printf("Public Key: %s\n", pubKey)
		fmt.Printf("File Path:  %s\n\n", config.Server.PrivateKeyPath)
		os.Exit(0)
	}

	// 5. Dial WebSocket with Pinned TLS
	dialer := websocket.Dialer{
		TLSClientConfig: internal.GetPinnedTLSConfig(config.Server.CertPin),
	}

	// Extract Module Names to report as Capabilities
	var capabilities []string
	for moduleName := range config.Modules {
		capabilities = append(capabilities, moduleName)
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

		// Authenticate
		helloPayload := map[string]interface{}{
			"public_key":   pubKey,
			"capabilities": capabilities, // Agent dynamically reports what modules it supports
		}
		conn.WriteJSON(WSPayload{Event: "AGENT_HELLO", Payload: helloPayload})

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

		// Execution Loop
		for {
			var msg WSPayload
			if err := conn.ReadJSON(&msg); err != nil {
				log.Println("Connection dropped or error:", err)
				break 
			}

			log.Printf("Received task: %s (ID: %s)", msg.Event, msg.TaskID)

			// The Event is now formatted as "module_name:action" (e.g., "system_user:create")
			parts := strings.Split(msg.Event, ":")
			if len(parts) != 2 {
				log.Printf("Error: Malformed event format '%s'. Expected 'module:action'", msg.Event)
				continue
			}
			moduleName := parts[0]
			actionName := parts[1]

			// Validate module exists
			module, moduleExists := config.Modules[moduleName]
			if !moduleExists {
				log.Printf("Error: Module '%s' not supported by this agent", moduleName)
				continue
			}

			// Validate action exists within the module
			actionConf, actionExists := module[actionName]
			if !actionExists {
				log.Printf("Error: Action '%s' not supported for module '%s'", actionName, moduleName)
				continue
			}

			// Execute Task
			outStr, execErr := internal.ExecuteTask(actionConf, msg.Payload)

			status := "SUCCESS"
			errorMsg := ""
			if execErr != nil {
				status = "FAILED"
				errorMsg = execErr.Error()
			}

			// Report Result
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

		conn.Close()
		log.Println("Disconnected from Central Server. Retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}