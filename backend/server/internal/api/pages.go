package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/config"
	"github.com/Sarin-jacob/Initiate/internal/crypto"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/markdown"
)

// --- ADMIN ENDPOINTS ---

type SavePageRequest struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type PagePreviewRequest struct {
	Content string `json:"content"`
}

func HandleListPages(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pages []db.Page
		database.Find(&pages)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pages)
	}
}

func HandleSavePage(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SavePageRequest
		json.NewDecoder(r.Body).Decode(&req)

		var page db.Page
		if err := database.Where("slug = ?", req.Slug).First(&page).Error; err != nil {
			page = db.Page{ID: uuid.New().String(), Slug: req.Slug}
		}

		page.Title = req.Title
		page.Content = req.Content

		database.Save(&page)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(page)
	}
}

// HandlePreviewPage processes markdown safely for the Admin editor
func HandlePreviewPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PagePreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		// Inject Mock Data so the Admin can preview how variables and loops look!
		mockData := markdown.OnboardingTemplateData{
			Username:  "JaneDoe",
			Email:     "jane.doe@company.com",
			GiteaURL:  config.App.GiteaExternalURL,
			SystemURL: config.App.BaseURL,
			Token:     "preview-token-xyz",
			InviteURL: fmt.Sprintf("%s/invite?token=preview-token-xyz", config.App.BaseURL),
			Servers: []markdown.ServerInfo{
				{Name: "production_db", Address: "10.0.0.50"},
				{Name: "edge_gateway", Address: "192.168.1.100"},
			},
		}

		renderedHTML, err := markdown.RenderGFM(req.Content, mockData)
		if err != nil {
			http.Error(w, "Markdown render failed. Check your {{ }} syntax.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"html_content": renderedHTML,
		})
	}
}

// --- USER ONBOARDING ENDPOINTS (Secured by Token) ---

func HandleGetPageBySlug(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawToken := chi.URLParam(r, "token")
		slug := chi.URLParam(r, "slug")

		var invite db.Invitation
		if err := database.Where("token_hash = ?", crypto.HashToken(rawToken)).First(&invite).Error; err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		if invite.UsedAt != nil || time.Now().After(invite.ExpiresAt) {
			http.Error(w, "Token expired", http.StatusUnauthorized)
			return
		}

		var user db.User
		database.First(&user, "id = ?", invite.UserID)

		var page db.Page
		if err := database.Where("slug = ?", slug).First(&page).Error; err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		// Look up actual assigned servers for the authenticated user
		var accesses []db.UserAccess
		database.Where("user_id = ?", user.ID).Find(&accesses)
		var assignedServers []markdown.ServerInfo
		for _, access := range accesses {
			if access.TargetType == "SERVER" {
				var srv db.TargetServer
				if err := database.First(&srv, "id = ?", access.TargetID).Error; err == nil {
					assignedServers = append(assignedServers, markdown.ServerInfo{Name: srv.Name, Address: srv.Address})
				}
			}
		}

		templateData := markdown.OnboardingTemplateData{
			Username:  user.Username,
			Email:     user.Email,
			GiteaURL:  config.App.GiteaExternalURL,
			SystemURL: config.App.BaseURL,
			Token:     rawToken,
			Servers:   assignedServers,
		}

		renderedHTML, _ := markdown.RenderGFM(page.Content, templateData)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"title":        page.Title,
			"html_content": renderedHTML,
		})
	}
}

// HandleGetPublicPage returns fully rendered HTML, requiring no auth
func HandleGetPublicPage(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		query := r.URL.Query()

		var page db.Page
		if err := database.Where("slug = ?", slug).First(&page).Error; err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		username := query.Get("username")
		if username == "" {
			username = "[Your Username]"
		}

		email := query.Get("email")
		if email == "" {
			email = "[Your Email]"
		}

		var servers []markdown.ServerInfo
		for _, paramVal := range query["server"] {
			var srv db.TargetServer
			if err := database.Where("name = ?", paramVal).First(&srv).Error; err == nil {
				servers = append(servers, markdown.ServerInfo{Name: srv.Name, Address: srv.Address})
			} else {
				servers = append(servers, markdown.ServerInfo{Name: "Server", Address: paramVal})
			}
		}

		if len(servers) == 0 {
			servers = append(servers, markdown.ServerInfo{Name: "Assigned Server", Address: "[Server's IP/Hostname]"})
		}

		templateData := markdown.OnboardingTemplateData{
			Username:  username,
			Email:     email,
			GiteaURL:  config.App.GiteaExternalURL,
			SystemURL: config.App.BaseURL,
			Servers:   servers,
		}

		renderedHTML, err := markdown.RenderGFM(page.Content, templateData)
		if err != nil {
			http.Error(w, "Template Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"title":        page.Title,
			"html_content": renderedHTML,
		})
	}
}

// HandleDeletePage safely removes a documentation/markdown page
func HandleDeletePage(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		// Edge Case 1: Empty slug
		if slug == "" {
			http.Error(w, "Slug parameter is missing", http.StatusBadRequest)
			return
		}

		// Edge Case 2: Ensure the page actually exists
		var page db.Page
		if err := database.Where("slug = ?", slug).First(&page).Error; err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		// Precaution: Check if this page is actively used in System Settings
		// We don't want admins breaking the onboarding flow by deleting active email/invite templates.
		var activeSettings []db.SystemSetting
		database.Where("value = ? AND key IN ?", slug, []string{
			"default_invite_slug", 
			"default_email_slug", 
			"welcome_email_slug",
		}).Find(&activeSettings)
		
		if len(activeSettings) > 0 {
			http.Error(w, "Cannot delete page: It is currently set as an active system template in your settings. Please change the system setting first.", http.StatusConflict)
			return
		}

		// Proceed with deletion
		if err := database.Delete(&page).Error; err != nil {
			http.Error(w, "Failed to delete page", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"message": "Page deleted successfully",
		})
	}
}