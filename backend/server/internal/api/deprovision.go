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
)

type TargetTeardownRequest struct {
	TargetID   string `json:"target_id"`
	PurgeHome  bool   `json:"purge_home,omitempty"`  // Used by system_user:delete
	PurgeRepos bool   `json:"purge_repos,omitempty"` // Used by gitea_user:delete
}

type DeprovisionRequest struct {
	Targets []TargetTeardownRequest `json:"targets"`
}

func HandleDeprovisionUser(database *gorm.DB, hub *agenthub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "id")
		
		var req DeprovisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
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

		for _, targetReq := range req.Targets {
			if !targetReq.PurgeHome && !targetReq.PurgeRepos {
				isFootprintPreserved = true // Hard Purge flags not set
			}

			var access db.UserAccess
			if err := database.Where("user_id = ? AND target_id = ?", user.ID, targetReq.TargetID).First(&access).Error; err == nil {
				
				if access.DeprovisionMacroID != "" {
					var macro db.Macro
					if err := database.First(&macro, "id = ?", access.DeprovisionMacroID).Error; err == nil {
						
						var steps []MacroStep
						json.Unmarshal([]byte(macro.Steps), &steps)

						payload := map[string]interface{}{
							"username":    user.Username,
							"purge_home":  targetReq.PurgeHome,
							"purge_repos": targetReq.PurgeRepos,
						}

						for _, step := range steps {
							event := fmt.Sprintf("%s:%s", step.Module, step.Action)
							log.Printf("Executing teardown step '%s' on %s...", event, targetReq.TargetID)
							hub.DispatchTaskSync(targetReq.TargetID, event, payload, 30*time.Second)
						}
					}
				}
				// Clean up the DB access record after running the teardown pipeline
				database.Delete(&access)
			}
		}

		var finalState string
		if isFootprintPreserved {
			user.Status = "ARCHIVED"
			database.Save(&user)
			finalState = "ARCHIVED"
		} else {
			database.Delete(&user)
			finalState = "PURGED"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":     user.ID,
			"final_state": finalState,
		})
	}
}