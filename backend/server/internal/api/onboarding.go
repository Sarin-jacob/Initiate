package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/crypto"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/gitea"
)

type CompleteOnboardingRequest struct {
	Password     string `json:"password"`
	SSHPublicKey string `json:"ssh_public_key"`
}

func HandleCompleteOnboarding(database *gorm.DB, hub *agenthub.Hub, giteaClient *gitea.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawToken := chi.URLParam(r, "token")
		if rawToken == "" {
			http.Error(w, "Missing invite token", http.StatusBadRequest)
			return
		}
		tokenHash := crypto.HashToken(rawToken)

		var req CompleteOnboardingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		var invite db.Invitation
		if err := database.Where("token_hash = ?", tokenHash).First(&invite).Error; err != nil {
			http.Error(w, "Invalid or expired invite token", http.StatusUnauthorized)
			return
		}
		if invite.UsedAt != nil || time.Now().After(invite.ExpiresAt) {
			http.Error(w, "This invite token has expired or already been used", http.StatusUnauthorized)
			return
		}

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

		for _, access := range accesses {
			if access.TargetType == "GITEA" {
				err := giteaClient.CreateUser(r.Context(), user.Username, user.Email, req.Password)
				if err != nil {
					log.Printf("Failed to provision Gitea for %s: %v", user.Username, err)
					database.Model(&access).Update("status", "FAILED")
				} else {
					database.Model(&access).Update("status", "ACTIVE")
				}
			}

			if access.TargetType == "SERVER" {
				payload := map[string]interface{}{
					"username":       user.Username,
					"password":       req.Password,
					"ssh_public_key": req.SSHPublicKey,
				}
				
				// 1. Parse the pipeline steps from the database
				var steps []MacroStep
				if access.GrantedModules != "" {
					// Attempt to unmarshal as the new strict Macro pipeline array
					if err := json.Unmarshal([]byte(access.GrantedModules), &steps); err != nil {
						// FALLBACK: If this is an older invite using the string array ["system_user"]
						var oldModules []string
						if fallbackErr := json.Unmarshal([]byte(access.GrantedModules), &oldModules); fallbackErr == nil {
							for _, mod := range oldModules {
								steps = append(steps, MacroStep{Module: mod, Action: "create"})
								if mod == "system_user" { // Preserve the old hardcoded rule for legacy invites
									steps = append(steps, MacroStep{Module: mod, Action: "set_password"})
								}
							}
						} else {
							log.Printf("Failed to decode pipeline steps for server %s: %v", access.TargetID, err)
							continue
						}
					}
				}

				// 2. Execute the Pipeline Synchronously
				pipelineSuccess := true
				
				for _, step := range steps {
					event := fmt.Sprintf("%s:%s", step.Module, step.Action)
					log.Printf("Executing %s on agent %s...", event, access.TargetID)

					// Block and wait up to 30 seconds for the Agent to complete this specific action
					res, err := hub.DispatchTaskSync(access.TargetID, event, payload, 30*time.Second)
					
					if err != nil {
						log.Printf("Pipeline aborted on %s: Timeout or connection lost during '%s'", access.TargetID, event)
						pipelineSuccess = false
						break // HALT PIPELINE
					}

					if res.Status == "FAILED" {
						log.Printf("Pipeline aborted on %s: Step '%s' failed. Error: %s", access.TargetID, event, res.Output)
						pipelineSuccess = false
						break // HALT PIPELINE
					}
					
					log.Printf("Step '%s' succeeded on %s", event, access.TargetID)
				}

				// 3. Update Database Status based on Pipeline success
				if pipelineSuccess {
					database.Model(&access).Update("status", "ACTIVE")
				} else {
					database.Model(&access).Update("status", "FAILED")
				}
			}
		}

		// Finalize the Onboarding (Mark invite used, update user)
		tx := database.Begin()
		now := time.Now()
		tx.Model(&invite).Update("used_at", now)
		tx.Model(&user).Updates(map[string]interface{}{
			"password_hash":  string(hashBytes),
			"ssh_public_key": req.SSHPublicKey,
			"status":         "ACTIVE",
		})
		tx.Commit()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Onboarding complete. Check dashboard for provisioning status.",
		})
	}
}