package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/db"
)

// MacroStep defines a single execution block in the pipeline
type MacroStep struct {
	Module string `json:"module"` // e.g., "system_user"
	Action string `json:"action"` // e.g., "create"
}

// MacroPayload is the JSON structure sent from the Svelte UI
type MacroPayload struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Steps       []MacroStep `json:"steps"`
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