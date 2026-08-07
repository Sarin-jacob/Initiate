package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/gitea"
)

type GiteaTeardownRequest struct {
	Enabled    bool `json:"enabled"`
	PurgeRepos bool `json:"purge_repos"`
}

type ServerTeardownRequest struct {
	TargetID  string `json:"target_id"`
	PurgeHome bool   `json:"purge_home"`
}

type DeprovisionRequest struct {
	Gitea   *GiteaTeardownRequest   `json:"gitea,omitempty"`
	Servers []ServerTeardownRequest `json:"servers"`
}

type DeprovisionResponse struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	FinalState string `json:"final_state"`
	Message    string `json:"message"`
}

func HandleDeprovisionUser(database *gorm.DB, hub *agenthub.Hub, giteaClient *gitea.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "id")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		var req DeprovisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		var user db.User
		if err := database.First(&user, "id = ?", userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		isFootprintPreserved := false

		// 1. Process Gitea Teardown
		if req.Gitea != nil && req.Gitea.Enabled {
			if !req.Gitea.PurgeRepos {
				isFootprintPreserved = true
				if err := giteaClient.DeleteUser(r.Context(), user.Username, false); err != nil {
					log.Printf("Gitea soft delete failed: %v", err)
				}
			} else {
				if err := giteaClient.DeleteUser(r.Context(), user.Username, true); err != nil {
					log.Printf("Gitea purge failed: %v", err)
				}
			}
			database.Where("user_id = ? AND target_type = ?", user.ID, "GITEA").Delete(&db.UserAccess{})
		}

		// 2. Process Edge Server Teardowns
		for _, srv := range req.Servers {
			if !srv.PurgeHome {
				isFootprintPreserved = true
			}

			// Fetch the user's specific access record to find out WHICH modules they have
			var access db.UserAccess
			if err := database.Where("user_id = ? AND target_type = ? AND target_id = ?", user.ID, "SERVER", srv.TargetID).First(&access).Error; err == nil {
				
				var modulesToTeardown []string
				
				// Decode the GrantedModules (handles both V1 flat arrays and V2 MacroSteps safely)
				if access.GrantedModules != "" {
					var flatModules []string
					if err := json.Unmarshal([]byte(access.GrantedModules), &flatModules); err == nil {
						modulesToTeardown = flatModules
					} else {
						var steps []MacroStep
						if err := json.Unmarshal([]byte(access.GrantedModules), &steps); err == nil {
							// Deduplicate modules from steps
							modMap := make(map[string]bool)
							for _, step := range steps {
								modMap[step.Module] = true
							}
							for m := range modMap {
								modulesToTeardown = append(modulesToTeardown, m)
							}
						}
					}
				}

				// Fallback safety net
				if len(modulesToTeardown) == 0 {
					modulesToTeardown = []string{"system_user"} 
				}

				payload := map[string]interface{}{
					"username":   user.Username,
					"purge_home": srv.PurgeHome,
				}

				// Synchronously execute the `:delete` action for every granted module
				for _, module := range modulesToTeardown {
					event := fmt.Sprintf("%s:delete", module)
					log.Printf("Executing teardown '%s' on %s...", event, srv.TargetID)
					
					res, err := hub.DispatchTaskSync(srv.TargetID, event, payload, 30*time.Second)
					
					if err != nil {
						log.Printf("Agent %s offline/timeout for '%s'. Manual cleanup may be required.", srv.TargetID, event)
					} else if res.Status == "FAILED" {
						log.Printf("Agent %s failed to execute '%s': %s", srv.TargetID, event, res.Output)
					} else {
						log.Printf("Successfully executed '%s' on %s", event, srv.TargetID)
					}
				}
			}

			// Clean up the DB access record regardless of agent online status
			database.Where("user_id = ? AND target_type = ? AND target_id = ?", user.ID, "SERVER", srv.TargetID).Delete(&db.UserAccess{})
		}

		// 3. Final State Decision
		var finalState string
		var message string

		if isFootprintPreserved {
			user.Status = "ARCHIVED"
			if err := database.Save(&user).Error; err != nil {
				http.Error(w, "Failed to update user status to ARCHIVED", http.StatusInternalServerError)
				return
			}
			finalState = "ARCHIVED"
			message = "Teardown complete. User status set to ARCHIVED because assets (/home or Gitea repos) were preserved."
		} else {
			if err := database.Delete(&user).Error; err != nil {
				http.Error(w, "Failed to purge user from database", http.StatusInternalServerError)
				return
			}
			finalState = "PURGED"
			message = "Teardown complete. User and all associated assets have been permanently purged."
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeprovisionResponse{
			UserID:     user.ID,
			Username:   user.Username,
			FinalState: finalState,
			Message:    message,
		})
	}
}