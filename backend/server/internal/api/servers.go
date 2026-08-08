package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/db"
)

type RegisterServerRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

type ServerConfigRequest struct {
	ProvisionMacroID       string `json:"provision_macro_id"`
	SoftDeprovisionMacroID string `json:"soft_deprovision_macro_id"`
	HardDeprovisionMacroID string `json:"hard_deprovision_macro_id"`
}

// HandleRegisterServer adds a new Edge Agent key to the database
func HandleRegisterServer(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterServerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		server := db.TargetServer{
			ID:        uuid.New().String(),
			Name:      req.Name,
			PublicKey: req.PublicKey,
			Status:    "OFFLINE", // Switches to ONLINE when it connects via WS
		}

		if err := database.Create(&server).Error; err != nil {
			http.Error(w, "Failed to register server. Name or Key might already exist.", http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Edge Server registered successfully",
			"id":      server.ID,
		})
	}
}

// HandleListServers returns all registered Edge Agents and their statuses
func HandleListServers(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var servers []db.TargetServer
		if err := database.Find(&servers).Error; err != nil {
			http.Error(w, "Failed to fetch servers", http.StatusInternalServerError)
			return
		}

		if servers == nil {
			servers = []db.TargetServer{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(servers)
	}
}

// HandleConfigServer saves the Macro bindings to the TargetServer
func HandleConfigServer(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverID := chi.URLParam(r, "id")
		
		var req ServerConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if err := database.Model(&db.TargetServer{}).Where("id = ?", serverID).Updates(map[string]interface{}{
			"provision_macro_id":        req.ProvisionMacroID,
			"soft_deprovision_macro_id": req.SoftDeprovisionMacroID,
			"hard_deprovision_macro_id": req.HardDeprovisionMacroID,
		}).Error; err != nil {
			http.Error(w, "Failed to update server configuration", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

// HandleDeleteServer safely deregisters an Edge Agent
func HandleDeleteServer(database *gorm.DB, hub *agenthub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverID := chi.URLParam(r, "id")
		
		if serverID == "internal-gitea" {
			http.Error(w, "Cannot deregister internal virtual systems", http.StatusForbidden)
			return
		}

		// Disconnect from websocket hub if active
		hub.DisconnectTarget(serverID) 

		if err := database.Delete(&db.TargetServer{}, "id = ?", serverID).Error; err != nil {
			http.Error(w, "Failed to deregister server", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}