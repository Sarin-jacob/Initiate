package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/db"
)

type DeprovisionRequest struct {
	PurgeRepos bool `json:"purge_repos"`
	PurgeHome  bool `json:"purge_home"`
}

func HandleDeprovisionUser(database *gorm.DB, hub *agenthub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "id")
		
		var req DeprovisionRequest
		json.NewDecoder(r.Body).Decode(&req)

		var user db.User
		if err := database.First(&user, "id = ?", userID).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		var accesses []db.UserAccess
		database.Where("user_id = ?", user.ID).Find(&accesses)

		hasFailures := false
		isFootprintPreserved := !req.PurgeHome && !req.PurgeRepos

		payload := map[string]interface{}{
			"username":    user.Username,
			"purge_home":  req.PurgeHome,
			"purge_repos": req.PurgeRepos,
		}

		for _, access := range accesses {
			var target db.TargetServer
			if err := database.First(&target, "id = ?", access.TargetID).Error; err != nil {
				continue
			}

			// Decide which macro the Agent requires based on UI flags
			macroID := target.SoftDeprovisionMacroID
			// If either purge flag is true, elevate to Hard Deprovision
			if req.PurgeHome || req.PurgeRepos {
				macroID = target.HardDeprovisionMacroID
			}

			if macroID != "" {
				var macro db.Macro
				if err := database.First(&macro, "id = ?", macroID).Error; err == nil {
					var steps []MacroStep
					json.Unmarshal([]byte(macro.Steps), &steps)

					targetSuccess := true
					for _, step := range steps {
						event := fmt.Sprintf("%s:%s", step.Module, step.Action)
						log.Printf("Executing teardown '%s' on %s...", event, target.ID)
						
						res, err := hub.DispatchTaskSync(target.ID, event, payload, 30*time.Second)
						if err != nil || res.Status == "FAILED" {
							log.Printf("CRITICAL: Teardown failed on %s (Step: %s). Output: %v", target.ID, event, res.Output)
							targetSuccess = false
							break // Halt this target's pipeline
						}
					}
					
					if !targetSuccess {
						hasFailures = true
						continue // Do NOT delete the UserAccess record if it failed!
					}
				}
			}

			// Pipeline succeeded (or none was configured), clean up access record
			database.Delete(&access)
		}

		var finalState string
		if hasFailures {
			// THE FAIL-SAFE: Do not delete the user. Mark them for manual intervention.
			database.Model(&user).Update("status", "DEPROVISION_FAILED")
			finalState = "DEPROVISION_FAILED"
		} else if isFootprintPreserved {
			database.Model(&user).Update("status", "ARCHIVED")
			finalState = "ARCHIVED"
		} else {
			database.Delete(&user)
			finalState = "PURGED"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":     user.ID,
			"final_state": finalState,
		})
	}
}