package api

import (
	"encoding/json"
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

// CompleteOnboardingRequest parses the data sent by the frontend form
type CompleteOnboardingRequest struct {
	Password     string `json:"password"`
	SSHPublicKey string `json:"ssh_public_key"`
}

// HandleCompleteOnboarding processes the final step of the user invite flow
func HandleCompleteOnboarding(database *gorm.DB, hub *agenthub.Hub, giteaClient *gitea.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract and hash the token from the URL
		rawToken := chi.URLParam(r, "token")
		if rawToken == "" {
			http.Error(w, "Missing invite token", http.StatusBadRequest)
			return
		}
		tokenHash := crypto.HashToken(rawToken)

		// 2. Parse payload (Password + SSH Key)
		var req CompleteOnboardingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		if len(req.Password) < 8 {
			http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		// 3. Database Validation Phase
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

		// 4. Hash the User's Password for the Web DB
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to process password", http.StatusInternalServerError)
			return
		}

		// 5. Fetch Provisioning Requirements
		var accesses []db.UserAccess
		database.Where("user_id = ?", user.ID).Find(&accesses)

		// 6. Execute External Provisioning
		for _, access := range accesses {
			if access.TargetType == "GITEA" {
				// Call Gitea REST API
				err := giteaClient.CreateUser(r.Context(), user.Username, user.Email, req.Password)
				if err != nil {
					log.Printf("Failed to provision Gitea for user %s: %v", user.Username, err)
				} else {
					database.Model(&access).Update("status", "ACTIVE")
				}
			}

			if access.TargetType == "SERVER" {
				// V2 Payload: Now includes the raw password so the Agent can run `chpasswd`
				payload := map[string]interface{}{
					"username":       user.Username,
					"password":       req.Password,
					"ssh_public_key": req.SSHPublicKey,
				}
				
				// Dispatch the 3-step V2 Sequence
				var agentErrs []error
				
				_, err1 := hub.DispatchTask(access.TargetID, "system_user:create", payload)
				if err1 != nil { agentErrs = append(agentErrs, err1) }
				
				_, err2 := hub.DispatchTask(access.TargetID, "system_user:set_password", payload)
				if err2 != nil { agentErrs = append(agentErrs, err2) }
				
				_, err3 := hub.DispatchTask(access.TargetID, "ssh_key:create", payload)
				if err3 != nil { agentErrs = append(agentErrs, err3) }

				if len(agentErrs) > 0 {
					log.Printf("Agent %s encountered errors provisioning %s: %v", access.TargetID, user.Username, agentErrs)
				} else {
					database.Model(&access).Update("status", "ACTIVE")
				}
			}
		}

		// 7. Finalize Database State (Transaction)
		tx := database.Begin()
		
		// Mark Invite as used
		now := time.Now()
		if err := tx.Model(&invite).Update("used_at", now).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Update User Record
		userUpdates := map[string]interface{}{
			"password_hash":  string(hashBytes),
			"ssh_public_key": req.SSHPublicKey,
			"status":         "ACTIVE",
		}
		if err := tx.Model(&user).Updates(userUpdates).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		tx.Commit()

		// 8. Respond Success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Onboarding complete. Your accounts have been provisioned.",
		})
	}
}