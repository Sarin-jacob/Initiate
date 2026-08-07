package api

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/gitea"
	"github.com/Sarin-jacob/Initiate/internal/mailer"
)

func NewRouter(
    database *gorm.DB, 
    hub *agenthub.Hub, 
    giteaClient *gitea.Client, 
    emailer *mailer.Mailer, 
    baseSystemURL string,
) *chi.Mux {
    r := chi.NewRouter()

    // Standard middlewares
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.RealIP)

    // --- 1. Admin API (Secured by JWT) ---
    r.Route("/api/admin", func(r chi.Router) {
        r.Use(RequireAdmin)
        
        r.Post("/users/invite", HandleInviteUser(database, emailer, baseSystemURL))
        r.Delete("/users/{id}", HandleDeprovisionUser(database, hub, giteaClient))
        
        // Stubs for future implementation
        r.Get("/users", HandleListUsers)
        r.Get("/servers", HandleListServers)
        r.Post("/servers", HandleRegisterServer)
    })

    // --- 2. Invite / Onboarding API (Unauthenticated initially, tokens checked inside handlers) ---
    r.Route("/api/invite", func(r chi.Router) {
        r.Get("/{token}", HandleGetInviteData(database))
        r.Post("/{token}/complete", HandleCompleteOnboarding(database, hub, giteaClient))
    })

    // --- 3. Edge Agent WebSocket (Secured by Ed25519) ---
    r.With(RequireEdgeAuth).Get("/agent/ws", hub.HandleWS)

    // --- 4. Frontend SPA Serve (Catch-all) ---
    fs := http.FileServer(http.Dir("./static"))
	
	// Catch-all route to support client-side routing (e.g. /invite?token=...)
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// If requesting a specific file that exists, serve it
		if _, err := os.Stat("./static" + r.URL.Path); err == nil {
			fs.ServeHTTP(w, r)
			return
		}
		// Otherwise, fallback to serving index.html (SPA routing behavior)
		http.ServeFile(w, r, "./static/index.html")
	})

    return r
}

// Stub handlers for endpoints not yet implemented
func HandleListUsers(w http.ResponseWriter, r *http.Request) { w.Write([]byte("List Users")) }
func HandleListServers(w http.ResponseWriter, r *http.Request) { w.Write([]byte("List Servers")) }
func HandleRegisterServer(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Register Server")) }
// func HandleServeFrontend(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Frontend UI")) }