package main

import (
    "crypto/ed25519"
    "encoding/hex"
    "log"

    "github.com/gorilla/websocket"
    "github.com/Sarin-jacob/Initiate-Agent/internal"
)

// WSPayload represents the standard JSON communication contract
type WSPayload struct {
    Event     string                 `json:"event"`
    TaskID    string                 `json:"task_id,omitempty"`
    Signature string                 `json:"signature,omitempty"`
    Nonce     string                 `json:"nonce,omitempty"`
    Payload   map[string]interface{} `json:"payload,omitempty"`
}

func main() {
    // 1. Load config
    config := internal.AgentConfig{} 
    config.Server.URL = "wss://192.168.1.50/agent/ws"
    config.Server.CertPin = "sha256:abcd..."
    
    // Load Keys
    privKey, err := internal.LoadPrivateKey("/etc/edgeauth/agent.key")
    if err != nil {
        log.Fatalf("Failed to load private key: %v", err)
    }
    pubKey := hex.EncodeToString(privKey.Public().(ed25519.PublicKey))

    // 2. Dial WebSocket with Pinned TLS
    dialer := websocket.Dialer{
        TLSClientConfig: internal.GetPinnedTLSConfig(config.Server.CertPin),
    }
    
    conn, _, err := dialer.Dial(config.Server.URL, nil)
    if err != nil {
        log.Fatalf("Failed to connect to Central Server: %v", err)
    }
    defer conn.Close()
    log.Println("Connected. Initiating Handshake...")

    // 3. Initiate Handshake
    hello := WSPayload{Event: "AGENT_HELLO", Payload: map[string]interface{}{"public_key": pubKey}}
    conn.WriteJSON(hello)

    var challenge WSPayload
    conn.ReadJSON(&challenge)
    if challenge.Event != "CHALLENGE" {
        log.Fatalf("Expected CHALLENGE, got %s", challenge.Event)
    }

    sig := internal.SignChallenge(privKey, challenge.Nonce)
    conn.WriteJSON(WSPayload{Event: "CHALLENGE_RESPONSE", Signature: sig})

    var authConfirm WSPayload
    conn.ReadJSON(&authConfirm)
    if authConfirm.Event != "AUTHENTICATED" {
        log.Fatalf("Authentication failed. Server responded: %s", authConfirm.Event)
    }
    log.Println("Successfully Authenticated. Listening for tasks...")

    // 4. Execution Loop
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

        result := WSPayload{
            Event:  "TASK_RESULT",
            TaskID: msg.TaskID,
            Payload: map[string]interface{}{
                "status": status,
                "output": outStr,
                "error":  errorMsg,
            },
        }
        conn.WriteJSON(result)
        log.Printf("Task %s completed with status %s", msg.TaskID, status)
    }
}