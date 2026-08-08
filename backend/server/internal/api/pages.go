package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

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
			GiteaURL:  os.Getenv("GITEA_EXTERNAL_URL"),
			SystemURL: os.Getenv("BASE_URL"),
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

// HandlePreviewPage accepts raw markdown, injects mock data, and returns rendered HTML
func HandlePreviewPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PagePreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		// Inject realistic mock data for the admin preview
		mockData := markdown.OnboardingTemplateData{
			Username:  "JaneDoe",
			Email:     "jane.doe@company.com",
			GiteaURL:  "https://gitea.example.com",
			SystemURL: "https://nexus.example.com",
			Token:     "preview-token-xyz-12345",
		}

		renderedHTML, err := markdown.RenderGFM(req.Content, mockData)
		if err != nil {
			http.Error(w, "Failed to render markdown preview", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"html_content": renderedHTML,
		})
	}
}

// HandleGetPublicPage returns a page without requiring JWT authentication
func HandleGetPublicPage(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		
		var page db.Page
		if err := database.Where("slug = ?", slug).First(&page).Error; err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	}
}