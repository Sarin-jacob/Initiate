package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/crypto"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/mailer"
	"github.com/Sarin-jacob/Initiate/internal/markdown"
)

type InviteUserRequest struct {
	Username         string   `json:"username"`
	Email            string   `json:"email"`
	ProvisionGitea   bool     `json:"provision_gitea"`
	EdgeServerIDs    []string `json:"edge_server_ids"`
	MarkdownTemplate string   `json:"markdown_template"`
}

// InviteDataResponse is the JSON payload sent to the frontend
type InviteDataResponse struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	HTMLContent string `json:"html_content"` // The fully rendered GFM
}

// HandleGetInviteData fetches the invite, injects variables, and renders the Markdown to HTML
func HandleGetInviteData(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Hash the token from the URL to check the DB
		rawToken := chi.URLParam(r, "token")
		if rawToken == "" {
			http.Error(w, "Missing invite token", http.StatusBadRequest)
			return
		}
		tokenHash := crypto.HashToken(rawToken)

		// 2. Look up the Invite and validate expiration
		var invite db.Invitation
		if err := database.Where("token_hash = ?", tokenHash).First(&invite).Error; err != nil {
			http.Error(w, "Invalid invite token", http.StatusNotFound)
			return
		}
		
		if invite.UsedAt != nil || time.Now().After(invite.ExpiresAt) {
			http.Error(w, "This invite token has expired or already been used", http.StatusUnauthorized)
			return
		}

		// 3. Look up the associated User
		var user db.User
		if err := database.First(&user, "id = ?", invite.UserID).Error; err != nil {
			http.Error(w, "Associated user not found", http.StatusInternalServerError)
			return
		}

		// 4. Prepare data for the Markdown template
		templateData := markdown.OnboardingTemplateData{
			Username:  user.Username,
			Email:     user.Email,
			GiteaURL:  os.Getenv("GITEA_EXTERNAL_URL"), // e.g., https://gitea.yourdomain.com
			SystemURL: os.Getenv("BASE_URL"),
		}

		// 5. Render the Markdown to HTML
		renderedHTML, err := markdown.RenderGFM(invite.MarkdownTemplate, templateData)
		if err != nil {
			http.Error(w, "Failed to render onboarding documentation", http.StatusInternalServerError)
			return
		}

		// 6. Return payload to frontend
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InviteDataResponse{
			Username:    user.Username,
			Email:       user.Email,
			HTMLContent: renderedHTML,
		})
	}
}

func HandleInviteUser(database *gorm.DB, emailer *mailer.Mailer, baseSystemURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InviteUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// 1. Generate secure token
		rawToken, tokenHash, err := crypto.GenerateInviteToken()
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// 2. Start a Database Transaction
		tx := database.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Create User (Status defaults to PENDING)
		userID := uuid.New().String()
		user := db.User{
			ID:       userID,
			Username: req.Username,
			Email:    req.Email,
			Status:   "PENDING",
		}
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to create user (username/email may already exist)", http.StatusConflict)
			return
		}

		// Create Access Records
		if req.ProvisionGitea {
			tx.Create(&db.UserAccess{UserID: userID, TargetType: "GITEA", TargetID: ""})
		}
		for _, serverID := range req.EdgeServerIDs {
			tx.Create(&db.UserAccess{UserID: userID, TargetType: "SERVER", TargetID: serverID})
		}

		// Create Invitation Record (Expires in 48 hours)
		expiresInHours := 48
		invite := db.Invitation{
			ID:               uuid.New().String(),
			UserID:           userID,
			TokenHash:        tokenHash, // Only storing the hash!
			MarkdownTemplate: req.MarkdownTemplate,
			ExpiresAt:        time.Now().Add(time.Duration(expiresInHours) * time.Hour),
		}
		if err := tx.Create(&invite).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to create invitation record", http.StatusInternalServerError)
			return
		}

		// 3. Dispatch Email
		inviteURL := fmt.Sprintf("%s/invite?token=%s", baseSystemURL, rawToken)
		fmt.Printf("\n\n=== TEST INVITE URL ===\n%s\n=======================\n\n", inviteURL)
		// if err := emailer.SendInvite(req.Email, req.Username, inviteURL, expiresInHours); err != nil {
		// 	tx.Rollback() // Rollback DB if email fails to send
		// 	http.Error(w, "Failed to send invitation email", http.StatusInternalServerError)
		// 	return
		// }

		// 4. Commit Transaction
		if err := tx.Commit().Error; err != nil {
			http.Error(w, "Database transaction failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "User invited and email dispatched",
			"user_id": userID,
		})
	}
}