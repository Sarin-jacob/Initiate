package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/crypto"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/mailer"
	"github.com/Sarin-jacob/Initiate/internal/markdown"
)

// Add a regex to find {{user.something}}
var userVarRegex = regexp.MustCompile(`\{\{user\.([a-zA-Z0-9_]+)\}\}`)

var adminVarRegex = regexp.MustCompile(`\{\{admin\.([a-zA-Z0-9_]+)\}\}`)

type InspectAdminVarsRequest struct {
	TargetIDs []string `json:"target_ids"`
}

type EdgeAllocation struct {
	ServerID string   `json:"server_id"`
	Modules  []string `json:"modules"`
}

type InviteUserRequest struct {
	Username     string   			`json:"username"`
	Email        string   			`json:"email"`
	ExpireAmount int      			`json:"expire_amount"`
	ExpireUnit   string   			`json:"expire_unit"`
	TargetIDs    []string 			`json:"target_ids"` // Simplest payload
	DocSlugs     []string 			`json:"doc_slugs"`  // Docs to inject
	AdminInputs  map[string]string 	`json:"admin_inputs"`
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
	RequiredVars []string `json:"required_vars"`
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

		templateData := markdown.OnboardingTemplateData{
			Username:  user.Username,
			Email:     user.Email,
			GiteaURL:  os.Getenv("GITEA_EXTERNAL_URL"),
			SystemURL: os.Getenv("BASE_URL"),
			Token:     rawToken,
		}

		renderedHTML, err := markdown.RenderGFM(invite.MarkdownTemplate, templateData)
		if err != nil {
			http.Error(w, "Failed to render onboarding documentation", http.StatusInternalServerError)
			return
		}
		// Dynamically fetch what this specific user needs to provide
		reqVars := getRequiredUserVars(database, user.ID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InviteDataResponse{
			Username:     user.Username,
			Email:        user.Email,
			HTMLContent:  renderedHTML,
			RequiredVars: reqVars, // Passed to Svelte
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
					markdownContent += fmt.Sprintf("- [%s](/api/invite/{{.Token}}/page/%s)\n", doc.Title, doc.Slug)
				}
			}
			docsJSON, _ := json.Marshal(req.DocSlugs)
			req.AdminInputs["_injected_docs"] = string(docsJSON)
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
		adminCtxBytes, _ := json.Marshal(req.AdminInputs)
		user := db.User{
			ID: userID, Username: req.Username, Email: req.Email, 
			Status: "PENDING", ExpiresAt: expiresAt,
			AdminContext: string(adminCtxBytes),
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
		var emailSlug db.SystemSetting
		database.Where("key = ?", "default_email_slug").First(&emailSlug)

		var emailPage db.Page
		subject := "Action Required: Complete your Server Onboarding"
		emailMD := "## Welcome {{.Username}}!\n\nAn administrator has provisioned access for you.\n\n[Click here to Complete Onboarding]({{.InviteURL}})"

		if emailSlug.Value != "" && database.Where("slug = ?", emailSlug.Value).First(&emailPage).Error == nil {
			subject = emailPage.Title // The Page Title becomes the Email Subject!
			emailMD = emailPage.Content
		}

		// 2. Render Markdown to HTML
		emailData := markdown.OnboardingTemplateData{
			Username:  user.Username,
			Email:     user.Email,
			SystemURL: baseSystemURL,
			Token:     rawToken,
			InviteURL: inviteURL,
		}
		renderedEmailHTML, _ := markdown.RenderGFM(emailMD, emailData)

		// 3. Inject Button Styling for the main Invite URL
		// We use string replacement to make the standard markdown link look like a beautiful button in the email client
		inviteHref := fmt.Sprintf(`href="%s"`, inviteURL)
		styledHref := fmt.Sprintf(`style="background-color: #2563eb; color: #ffffff; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block; font-weight: bold; margin: 16px 0;" href="%s"`, inviteURL)
		
		renderedEmailHTML = strings.ReplaceAll(renderedEmailHTML, inviteHref, styledHref)

		// 4. Send Email
		if emailer != nil {
			go emailer.SendHTML(user.Email, subject, renderedEmailHTML)
		}

		tx.Commit()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "user_id": userID})
	}
}

func HandleGetAdminVars(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InspectAdminVarsRequest
		json.NewDecoder(r.Body).Decode(&req)

		varSet := make(map[string]bool)
		for _, tid := range req.TargetIDs {
			var srv db.TargetServer
			if err := database.First(&srv, "id = ?", tid).Error; err == nil && srv.ProvisionMacroID != "" {
				var mac db.Macro
				if err := database.First(&mac, "id = ?", srv.ProvisionMacroID).Error; err == nil {
					matches := adminVarRegex.FindAllStringSubmatch(mac.Steps, -1)
					for _, m := range matches {
						if len(m) > 1 {
							varSet[m[1]] = true // Store unique admin var names
						}
					}
				}
			}
		}

		var reqVars []string
		for k := range varSet {
			reqVars = append(reqVars, k)
		}
		if reqVars == nil {
			reqVars = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"admin_vars": reqVars})
	}
}

// Helper function to scan assigned macros
func getRequiredUserVars(database *gorm.DB, userID string) []string {
	var accesses []db.UserAccess
	database.Where("user_id = ?", userID).Find(&accesses)

	varSet := make(map[string]bool)

	for _, acc := range accesses {
		if acc.TargetType == "SERVER" {
			var srv db.TargetServer
			// Find the server and its attached macro
			if err := database.First(&srv, "id = ?", acc.TargetID).Error; err == nil && srv.ProvisionMacroID != "" {
				var mac db.Macro
				if err := database.First(&mac, "id = ?", srv.ProvisionMacroID).Error; err == nil {
					// Scan the raw JSON string for {{user.xyz}} bindings
					matches := userVarRegex.FindAllStringSubmatch(mac.Steps, -1)
					for _, m := range matches {
						if len(m) > 1 {
							varSet[m[1]] = true
						}
					}
				}
			}
		}
	}

	var reqVars []string
	for k := range varSet {
		reqVars = append(reqVars, k)
	}
	return reqVars
}