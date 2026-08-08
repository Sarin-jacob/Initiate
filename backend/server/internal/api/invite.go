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

type EdgeAllocation struct {
	ServerID string   `json:"server_id"`
	Modules  []string `json:"modules"`
}

type InviteUserRequest struct {
	Username     string             `json:"username"`
	Email        string             `json:"email"`
	Allocations  []TargetAllocation `json:"allocations"` // Unifies Gitea and Servers
	ExpireAmount int                `json:"expire_amount"`
	ExpireUnit   string             `json:"expire_unit"`
}

type TargetAllocation struct {
	TargetID           string `json:"target_id"`            // e.g., "internal-gitea" or "edge-uuid"
	TargetType         string `json:"target_type"`          // "GITEA" or "SERVER"
	ProvisionMacroID   string `json:"provision_macro_id"`   // NEW
	DeprovisionMacroID string `json:"deprovision_macro_id"` // NEW
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
		rawToken := chi.URLParam(r, "token")
		if rawToken == "" {
			http.Error(w, "Missing invite token", http.StatusBadRequest)
			return
		}
		tokenHash := crypto.HashToken(rawToken)

		var invite db.Invitation
		if err := database.Where("token_hash = ?", tokenHash).First(&invite).Error; err != nil {
			http.Error(w, "Invalid invite token", http.StatusNotFound)
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

		// Prepare data for the Markdown template
		templateData := markdown.OnboardingTemplateData{
			Username:  user.Username,
			Email:     user.Email,
			GiteaURL:  os.Getenv("GITEA_EXTERNAL_URL"), 
			SystemURL: os.Getenv("BASE_URL"),
			Token:     rawToken, // CRITICAL: Added so CMS pages can create links like /page/{{.Token}}/ssh
		}

		renderedHTML, err := markdown.RenderGFM(invite.MarkdownTemplate, templateData)
		if err != nil {
			http.Error(w, "Failed to render onboarding documentation", http.StatusInternalServerError)
			return
		}

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

		// 2. Fetch the default CMS page for Onboarding (RESTORED)
		var defaultSlugSetting db.SystemSetting
		database.Where("key = ?", "default_invite_slug").First(&defaultSlugSetting)
		
		var defaultPage db.Page
		markdownContent := "## Welcome {{.Username}}!\n\nPlease set your password below to finalize your account."
		if defaultSlugSetting.Value != "" {
			if err := database.Where("slug = ?", defaultSlugSetting.Value).First(&defaultPage).Error; err == nil {
				markdownContent = defaultPage.Content
			}
		}

		var expiresAt *time.Time
		if req.ExpireAmount > 0 {
			expTime := time.Now()
			switch req.ExpireUnit {
			case "days":
				expTime = expTime.AddDate(0, 0, req.ExpireAmount)
			case "weeks":
				expTime = expTime.AddDate(0, 0, req.ExpireAmount*7)
			case "months":
				expTime = expTime.AddDate(0, req.ExpireAmount, 0)
			case "years":
				expTime = expTime.AddDate(req.ExpireAmount, 0, 0)
			default:
				expTime = expTime.AddDate(0, 0, req.ExpireAmount) // Default to days
			}
			expiresAt = &expTime
		}

		// 3. Start a Database Transaction
		tx := database.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Create User (Status defaults to PENDING)
		userID := uuid.New().String()
		user := db.User{
			ID:        userID,
			Username:  req.Username,
			Email:     req.Email,
			Status:    "PENDING",
			ExpiresAt: expiresAt,
		}
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to create user (username/email may already exist)", http.StatusConflict)
			return
		}
		
		// NEW: Map Unified Access (Handles both Gitea and Edge Servers identically)
		for _, alloc := range req.Allocations {
			tx.Create(&db.UserAccess{
				ID:                 uuid.New().String(),
				UserID:             userID,
				TargetType:         alloc.TargetType,
				TargetID:           alloc.TargetID,
				ProvisionMacroID:   alloc.ProvisionMacroID,
				DeprovisionMacroID: alloc.DeprovisionMacroID,
				Status:             "PENDING", // Remains pending until they complete onboarding
			})
		}

		// Create Invitation Record (Expires in 48 hours)
		expiresInHours := 48
		invite := db.Invitation{
			ID:               uuid.New().String(),
			UserID:           userID,
			TokenHash:        tokenHash, 
			MarkdownTemplate: markdownContent, // Sourced from CMS
			ExpiresAt:        time.Now().Add(time.Duration(expiresInHours) * time.Hour),
		}
		if err := tx.Create(&invite).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to create invitation record", http.StatusInternalServerError)
			return
		}

		// 4. Dispatch Email
		inviteURL := fmt.Sprintf("%s/invite?token=%s", baseSystemURL, rawToken)
		if err := emailer.SendInvite(req.Email, req.Username, inviteURL, expiresInHours); err != nil {
			tx.Rollback() 
			http.Error(w, "Failed to send invitation email", http.StatusInternalServerError)
			return
		}

		// 5. Commit Transaction
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