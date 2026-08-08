package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/gitea"
)

type CompleteOnboardingRequest struct {
	Password     string `json:"password"`
	SSHPublicKey string `json:"ssh_public_key"`
}

func HandleCompleteOnboarding(database *gorm.DB, hub *agenthub.Hub, giteaClient *gitea.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CompleteOnboardingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		invite := r.Context().Value("invite").(db.Invitation)

		var user db.User
		if err := database.First(&user, "id = ?", invite.UserID).Error; err != nil {
			http.Error(w, "Associated user not found", http.StatusInternalServerError)
			return
		}

		hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to process password", http.StatusInternalServerError)
			return
		}

		var accesses []db.UserAccess
		database.Where("user_id = ?", user.ID).Find(&accesses)

		payload := map[string]interface{}{
			"username":       user.Username,
			"password":       req.Password,
			"email":          user.Email, // Needed for Gitea Virtual Agent
			"ssh_public_key": req.SSHPublicKey,
		}

		// UNIFIED PIPELINE EXECUTION
		for _, access := range accesses {
			// Look up the actual Server/Virtual Agent to find out what its macro is
			var target db.TargetServer
			if err := database.First(&target, "id = ?", access.TargetID).Error; err != nil {
				continue
			}

			if target.ProvisionMacroID == "" {
				database.Model(&access).Update("status", "ACTIVE")
				continue // Agent opts out of automated provisioning
			}

			var macro db.Macro
			if err := database.First(&macro, "id = ?", target.ProvisionMacroID).Error; err != nil {
				database.Model(&access).Update("status", "FAILED")
				continue
			}

			var steps []MacroStep
			json.Unmarshal([]byte(macro.Steps), &steps)

			pipelineSuccess := true
			for _, step := range steps {
				event := fmt.Sprintf("%s:%s", step.Module, step.Action)
				log.Printf("Executing %s on target %s...", event, target.ID)

				res, err := hub.DispatchTaskSync(target.ID, event, payload, 30*time.Second)
				if err != nil || res.Status == "FAILED" {
					log.Printf("Pipeline aborted on %s: Step '%s' failed.", target.ID, event)
					pipelineSuccess = false
					break
				}
			}

			if pipelineSuccess {
				database.Model(&access).Update("status", "ACTIVE")
			} else {
				database.Model(&access).Update("status", "FAILED")
			}
		}

		// Finalize Onboarding
		tx := database.Begin()
		tx.Model(&invite).Update("used_at", time.Now())
		tx.Model(&user).Updates(map[string]interface{}{
			"password_hash":  string(hashBytes),
			"ssh_public_key": req.SSHPublicKey,
			"status":         "ACTIVE",
		})
		tx.Commit()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Provisioning pipelines completed."})
	}
}