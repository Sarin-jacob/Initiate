package agenthub

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// DispatchTask sends a command payload to a specific Edge Agent
func (h *Hub) DispatchTask(serverID string, event string, payload map[string]interface{}) (string, error) {
	h.mu.RLock()
	client, exists := h.clients[serverID]
	h.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("agent %s is currently offline", serverID)
	}

	taskID := uuid.New().String()
	task := WSPayload{
		Event:   event,
		TaskID:  taskID,
		Payload: payload,
	}

	// Write payload to WebSocket
	if err := client.Conn.WriteJSON(task); err != nil {
		h.unregisterClient(serverID)
		return "", fmt.Errorf("failed to send task to agent: %w", err)
	}

	log.Printf("Dispatched task %s (%s) to agent %s", taskID, event, serverID)
	return taskID, nil
}

// listen keeps the WebSocket alive and processes async task results
func (h *Hub) listen(client *AgentClient) {
	defer h.unregisterClient(client.ServerID)

	for {
		var msg WSPayload
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			break // Connection closed or errored
		}

		if msg.Event == "TASK_RESULT" {
			status, _ := msg.Payload["status"].(string)
			output, _ := msg.Payload["output"].(string)
			
			log.Printf("[Agent %s] Task %s Result: %s", client.ServerID, msg.TaskID, status)
			
			// If execution failed, log the output for debugging
			if status == "FAILED" {
				log.Printf("[Agent %s] Task %s Output/Error: %s", client.ServerID, msg.TaskID, output)
			}
			
			// TODO: You could save this task result to a `job_logs` database table here for audit trails.
		}
	}
}