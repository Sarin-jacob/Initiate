package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/db"
)

// MacroStep defines a single execution block in the pipeline
type MacroStep struct {
	Module string            `json:"module"`
	Action string            `json:"action"`
	Params map[string]string `json:"params"` // NEW: Captures the UI Parameter Bindings!
}

// MacroPayload is the JSON structure sent from the Svelte UI
type MacroPayload struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Steps       []MacroStep `json:"steps"`
}

type ApplyMacroRequest struct {
	MacroID  string `json:"macro_id"`
	ServerID string `json:"server_id"`
}

// HandleGetMacros returns all saved Macros
func HandleGetMacros(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var macros []db.Macro
		if err := database.Find(&macros).Error; err != nil {
			http.Error(w, "Failed to load macros", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(macros)
	}
}

// HandleCreateMacro saves a new ordered pipeline
func HandleCreateMacro(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload MacroPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if len(payload.Steps) == 0 {
			http.Error(w, "A macro must contain at least one step", http.StatusBadRequest)
			return
		}

		// Convert the ordered steps into a JSON string
		stepsJSON, err := json.Marshal(payload.Steps)
		if err != nil {
			http.Error(w, "Failed to encode macro steps", http.StatusInternalServerError)
			return
		}

		macro := db.Macro{
			ID:          uuid.New().String(),
			Name:        payload.Name,
			Description: payload.Description,
			Steps:       string(stepsJSON),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := database.Create(&macro).Error; err != nil {
			http.Error(w, "Failed to save macro. Name must be unique.", http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(macro)
	}
}

// HandleUpdateMacro processes edits to an existing pipeline
func HandleUpdateMacro(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		macroID := chi.URLParam(r, "id")
		if macroID == "" {
			http.Error(w, "Macro ID is required", http.StatusBadRequest)
			return
		}

		var payload MacroPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if len(payload.Steps) == 0 {
			http.Error(w, "A macro must contain at least one step", http.StatusBadRequest)
			return
		}

		var macro db.Macro
		if err := database.First(&macro, "id = ?", macroID).Error; err != nil {
			http.Error(w, "Macro not found", http.StatusNotFound)
			return
		}

		stepsJSON, err := json.Marshal(payload.Steps)
		if err != nil {
			http.Error(w, "Failed to encode macro steps", http.StatusInternalServerError)
			return
		}

		// Update fields
		macro.Name = payload.Name
		macro.Description = payload.Description
		macro.Steps = string(stepsJSON)
		macro.UpdatedAt = time.Now()

		if err := database.Save(&macro).Error; err != nil {
			http.Error(w, "Failed to update macro. Name must remain unique.", http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(macro)
	}
}

// HandleDeleteMacro completely removes a pipeline
func HandleDeleteMacro(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		macroID := chi.URLParam(r, "id")
		if macroID == "" {
			http.Error(w, "Macro ID is required", http.StatusBadRequest)
			return
		}

		if err := database.Delete(&db.Macro{}, "id = ?", macroID).Error; err != nil {
			http.Error(w, "Failed to delete macro", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Macro deleted successfully",
		})
	}
}

// HandleApplyMacro executes a saved pipeline for an existing user
func HandleApplyMacro(database *gorm.DB, hub *agenthub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "id")
		
		var req ApplyMacroRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		var user db.User
		if err := database.First(&user, "id = ?", userID).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		var macro db.Macro
		if err := database.First(&macro, "id = ?", req.MacroID).Error; err != nil {
			http.Error(w, "Macro not found", http.StatusNotFound)
			return
		}

		var steps []MacroStep // Ensure this struct is available in this file!
		if err := json.Unmarshal([]byte(macro.Steps), &steps); err != nil {
			http.Error(w, "Invalid macro JSON", http.StatusInternalServerError)
			return
		}

		// BUILD V3 RESOLVER PAYLOAD
		payload := map[string]interface{}{
			"sys.username": user.Username,
			"sys.email":    user.Email,
			"sys.id":       user.ID,
		}
		if user.AdminContext != "" {
			var adminCtx map[string]string
			json.Unmarshal([]byte(user.AdminContext), &adminCtx)
			for k, v := range adminCtx {
				payload[fmt.Sprintf("admin.%s", k)] = v
			}
		}

		// FIND THE ACCESS RECORD TO SAVE LOGS
		var access db.UserAccess
		database.Where("user_id = ? AND target_id = ?", user.ID, req.ServerID).First(&access)

		var executionTrace string
		pipelineSuccess := true

		for _, step := range steps {
			event := fmt.Sprintf("%s:%s", step.Module, step.Action)
			executionTrace += fmt.Sprintf("[PENDING] Manual trigger: %s...\n", event)

			resolvedPayload := resolveStepParams(step.Params, payload) // Uses the V3 resolver
			res, err := hub.DispatchTaskSync(req.ServerID, event, resolvedPayload, 30*time.Second)
			
			if err != nil {
				pipelineSuccess = false
				executionTrace += fmt.Sprintf("[ERROR] Network: %v\n", err)
				break
			}
			if res.Status == "FAILED" {
				pipelineSuccess = false
				executionTrace += fmt.Sprintf("[FAILED] Output:\n%s\n", res.Output)
				break
			}
			executionTrace += fmt.Sprintf("[SUCCESS] Output:\n%s\n", res.Output)
		}

		// Save Execution Logs
		if access.ID != "" {
			updates := map[string]interface{}{"execution_log": executionTrace}
			if !pipelineSuccess {
				updates["status"] = "FAILED"
			} else {
				updates["status"] = "ACTIVE"
			}
			database.Model(&access).Updates(updates)
		}

		if !pipelineSuccess {
			http.Error(w, "Macro execution failed. Check logs.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}