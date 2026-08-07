package api

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/db"
)

// HandleGetSettings returns all system settings as a simple key-value map
func HandleGetSettings(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var settings []db.SystemSetting
		if err := database.Find(&settings).Error; err != nil {
			http.Error(w, "Failed to load settings", http.StatusInternalServerError)
			return
		}

		// Convert database rows into a flat JSON object
		configMap := make(map[string]string)
		for _, s := range settings {
			configMap[s.Key] = s.Value
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(configMap)
	}
}

// HandleUpdateSettings accepts a JSON map and upserts the values into the database
func HandleUpdateSettings(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		// Run updates inside a transaction
		tx := database.Begin()
		for key, val := range payload {
			setting := db.SystemSetting{
				Key:   key,
				Value: val,
			}
			// Save performs an upsert by primary key
			if err := tx.Save(&setting).Error; err != nil {
				tx.Rollback()
				http.Error(w, "Failed to save settings", http.StatusInternalServerError)
				return
			}
		}
		tx.Commit()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Settings updated successfully",
		})
	}
}