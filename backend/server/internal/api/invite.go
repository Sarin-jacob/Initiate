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
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	ExpireAmount int      `json:"expire_amount"`
	ExpireUnit   string   `json:"expire_unit"`
	TargetIDs    []string `json:"target_ids"` // Simplest payload
	DocSlugs     []string `json:"doc_slugs"`  // Docs to inject
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
		invite := r.Context().Value("invite").(db.Invitation)
		rawToken := chi.URLParam(r, "token")

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
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		rawToken, tokenHash, _ := crypto.GenerateInviteToken()

		// 1. Prepare CMS Markdown
		var defaultSlug db.SystemSetting
		database.Where("key = ?", "default_invite_slug").First(&defaultSlug)
		
		markdownContent := "## Welcome {{.Username}}!\n\nPlease set your password below."
		var defaultPage db.Page
		if defaultSlug.Value != "" && database.Where("slug = ?", defaultSlug.Value).First(&defaultPage).Error == nil {
			markdownContent = defaultPage.Content
		}

		// Inject selected documentation directly into the markdown payload
		if len(req.DocSlugs) > 0 {
			markdownContent += "\n\n---\n### Attached Documentation:\n"
			for _, slug := range req.DocSlugs {
				var doc db.Page
				if database.Where("slug = ?", slug).First(&doc).Error == nil {
					// The frontend router intercepts /api/invite/.../page/ URLs!
					markdownContent += fmt.Sprintf("* [%s](/api/invite/{{.Token}}/page/%s)\n", doc.Title, doc.Slug)
				}
			}
		}

		// 2. Math for Expiration
		var expiresAt *time.Time
		if req.ExpireAmount > 0 {
			expTime := time.Now()
			switch req.ExpireUnit {
			case "days": expTime = expTime.AddDate(0, 0, req.ExpireAmount)
			case "weeks": expTime = expTime.AddDate(0, 0, req.ExpireAmount*7)
			case "months": expTime = expTime.AddDate(0, req.ExpireAmount, 0)
			case "years": expTime = expTime.AddDate(req.ExpireAmount, 0, 0)
			default: expTime = expTime.AddDate(0, 0, req.ExpireAmount)
			}
			expiresAt = &expTime
		}

		tx := database.Begin()
		defer func() {
			if r := recover(); r != nil { tx.Rollback() }
		}()

		userID := uuid.New().String()
		user := db.User{
			ID: userID, Username: req.Username, Email: req.Email, 
			Status: "PENDING", ExpiresAt: expiresAt,
		}
		if tx.Create(&user).Error != nil {
			tx.Rollback(); http.Error(w, "Username/Email conflict", http.StatusConflict); return
		}

		// Map basic target access
		for _, targetID := range req.TargetIDs {
			targetType := "SERVER"
			if targetID == "internal-gitea" { targetType = "GITEA" }
			tx.Create(&db.UserAccess{
				ID: uuid.New().String(), UserID: userID, TargetType: targetType, TargetID: targetID,
			})
		}

		invite := db.Invitation{
			ID: uuid.New().String(), UserID: userID, TokenHash: tokenHash, 
			MarkdownTemplate: markdownContent, ExpiresAt: time.Now().Add(48 * time.Hour),
		}
		tx.Create(&invite)

		// Dispatch email
		inviteURL := fmt.Sprintf("%s/invite?token=%s", baseSystemURL, rawToken)
		if emailer.SendInvite(req.Email, req.Username, inviteURL, 48) != nil {
			tx.Rollback(); http.Error(w, "Email failed", http.StatusInternalServerError); return
		}

		tx.Commit()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "user_id": userID})
	}
}