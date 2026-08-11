package api

import (
	"encoding/json"
	"net/http"
	"strings"
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

		// Upsert logic based on Slug
		var page db.Page
		if err := database.Where("slug = ?", req.Slug).First(&page).Error; err != nil {
			// Create new
			page = db.Page{ID: uuid.New().String(), Slug: req.Slug}
		}

		page.Title = req.Title
		page.Content = req.Content

		database.Save(&page)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(page)
	}
}

// --- USER ONBOARDING ENDPOINTS (Secured by Token) ---

func HandleGetPageBySlug(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawToken := chi.URLParam(r, "token")
		slug := chi.URLParam(r, "slug")

		// 1. Verify token exists and is valid
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

		// 2. Fetch the requested page
		var page db.Page
		if err := database.Where("slug = ?", slug).First(&page).Error; err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		// 3. Render Markdown safely
		templateData := markdown.OnboardingTemplateData{
			Username:  user.Username,
			Email:     user.Email,
			GiteaURL:  config.App.GiteaExternalURL,
			SystemURL: config.App.BaseURL,
			Token:     rawToken,
		}

		renderedHTML, _ := markdown.RenderGFM(page.Content, templateData)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"title":        page.Title,
			"html_content": renderedHTML,
		})
	}
}

// sanitizeTemplate prevents Go's text/template from crashing on unknown variables
func sanitizeTemplate(content string, isPreview bool) string {
	// 1. Swap known variables with mock data or placeholders
	if isPreview {
		content = strings.ReplaceAll(content, "{{.Username}}", "JaneDoe")
		content = strings.ReplaceAll(content, "{{.Email}}", "jane.doe@company.com")
		content = strings.ReplaceAll(content, "{{.GiteaURL}}", "https://gitea.example.com")
		content = strings.ReplaceAll(content, "{{.SystemURL}}", "https://nexus.example.com")
		content = strings.ReplaceAll(content, "{{.Token}}", "preview-token-xyz")
		content = strings.ReplaceAll(content, "{{.InviteURL}}", "https://nexus.example.com/invite?token=preview")
	} else {
		// Public Docs don't have user context, so we show placeholders
		content = strings.ReplaceAll(content, "{{.Username}}", "[Your Username]")
		content = strings.ReplaceAll(content, "{{.Email}}", "[Your Email]")
		content = strings.ReplaceAll(content, "{{.GiteaURL}}", "[Gitea URL]")
		content = strings.ReplaceAll(content, "{{.SystemURL}}", "[System URL]")
	}

	// 2. Escape any remaining {{ }} so the Go template engine treats them as literal strings
	content = strings.ReplaceAll(content, "{{", "&#123;&#123;")
	content = strings.ReplaceAll(content, "}}", "&#125;&#125;")
	
	return content
}

// HandlePreviewPage processes markdown safely for the Admin editor
func HandlePreviewPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PagePreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		safeContent := sanitizeTemplate(req.Content, true)
		
		// Pass nil data since we already injected the strings safely
		renderedHTML, err := markdown.RenderGFM(safeContent, nil) 
		if err != nil {
			http.Error(w, "Markdown render failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"html_content": renderedHTML,
		})
	}
}

// HandleGetPublicPage returns fully rendered HTML, requiring no auth
func HandleGetPublicPage(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		
		var page db.Page
		if err := database.Where("slug = ?", slug).First(&page).Error; err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		safeContent := sanitizeTemplate(page.Content, false)
		renderedHTML, _ := markdown.RenderGFM(safeContent, nil)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"title":        page.Title,
			"html_content": renderedHTML,
		})
	}
}