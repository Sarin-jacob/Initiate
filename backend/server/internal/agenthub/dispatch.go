package agenthub

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// DispatchTaskSync sends a command and blocks until the agent replies or it times out.
func (h *Hub) DispatchTaskSync(serverID string, event string, payload map[string]interface{}, timeout time.Duration) (TaskResult, error) {
	taskID := uuid.New().String()
	
	if serverID == "internal-gitea" {
		log.Printf("Intercepted %s for Virtual Agent (Gitea)", event)
		return h.handleVirtualGiteaTask(taskID, event, payload), nil
	}

	h.mu.RLock()
	client, exists := h.clients[serverID]
	h.mu.RUnlock()

	if !exists {
		return TaskResult{}, fmt.Errorf("agent %s is offline", serverID)
	}

	task := WSPayload{
		Event:   event,
		TaskID:  taskID,
		Payload: payload,
	}

	// 1. Create a buffered channel for this specific TaskID
	resultChan := make(chan TaskResult, 1)

	// 2. Register the channel in the Hub's pending tasks registry
	h.taskMu.Lock()
	h.pendingTasks[taskID] = resultChan
	h.taskMu.Unlock()

	// 3. Ensure we clean up the channel registry no matter how this function exits
	defer func() {
		h.taskMu.Lock()
		delete(h.pendingTasks, taskID)
		h.taskMu.Unlock()
		close(resultChan)
	}()

	// 4. Dispatch the payload over the wire
	if err := client.Conn.WriteJSON(task); err != nil {
		h.unregisterClient(serverID)
		return TaskResult{}, fmt.Errorf("failed to write to websocket: %w", err)
	}

	log.Printf("Dispatched SYNC task %s (%s) to agent %s", taskID, event, serverID)

	// 5. Block and wait! 
	// The goroutine stops here until data hits resultChan OR the timeout hits.
	select {
	case res := <-resultChan:
		return res, nil
	case <-time.After(timeout):
		return TaskResult{}, fmt.Errorf("task %s timed out after %v", taskID, timeout)
	}
}

// listen keeps the WebSocket alive and routes incoming results to waiting channels
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
			if status == "FAILED" {
				log.Printf("[Agent %s] Task %s Output/Error: %s", client.ServerID, msg.TaskID, output)
			}

			// NEW: Intercept the result and route it to the waiting goroutine
			h.taskMu.RLock()
			waitingChan, exists := h.pendingTasks[msg.TaskID]
			h.taskMu.RUnlock()

			if exists {
				// Use a non-blocking send in case the waiting function timed out 
				// a microsecond before this arrived.
				select {
				case waitingChan <- TaskResult{TaskID: msg.TaskID, Status: status, Output: output}:
					// Routed successfully!
				default:
					log.Printf("Warning: Dropped result for task %s (channel closed/full)", msg.TaskID)
				}
			}
		}
	}
}

// handleVirtualGiteaTask translates macro events into local REST API calls
func (h *Hub) handleVirtualGiteaTask(taskID string, event string, payload map[string]interface{}) TaskResult {
	username, _ := payload["username"].(string)
	
	var err error
	switch event {
	case "gitea_user:create":
		email, _ := payload["email"].(string) 
		password, _ := payload["password"].(string)
		err = h.gitea.CreateUser(context.Background(), username, email, password)
	
	case "gitea_user:delete":
		purgeRepos, _ := payload["purge_repos"].(bool)
		err = h.gitea.DeleteUser(context.Background(), username, purgeRepos)
	
	// Add cases for suspend/resume later as needed
	default:
		return TaskResult{TaskID: taskID, Status: "FAILED", Output: "Unknown Gitea action: " + event}
	}

	if err != nil {
		return TaskResult{TaskID: taskID, Status: "FAILED", Output: err.Error()}
	}
	return TaskResult{TaskID: taskID, Status: "SUCCESS", Output: "Gitea task completed successfully"}
}