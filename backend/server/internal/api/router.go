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
        r.Delete("/users/{id}", HandleDeprovisionUser(database, hub))

        r.Get("/pages", HandleListPages(database))
        r.Post("/pages", HandleSavePage(database))
        r.Post("/pages/preview", HandlePreviewPage())
        
        // Stubs for future implementation
        r.Get("/servers", HandleListServers(database))
        r.Post("/servers", HandleRegisterServer(database))
        r.Put("/servers/{id}/config", HandleConfigServer(database))
        r.Delete("/servers/{id}", HandleDeleteServer(database, hub))
        
        r.Get("/settings", HandleGetSettings(database))
        r.Post("/settings", HandleUpdateSettings(database))
        
        r.Get("/users", HandleListUsers(database))
        r.Put("/users/{id}/expire", HandleUpdateUserExpiration(database))
        r.Post("/users/{id}/macro", HandleApplyMacro(database, hub))
        r.Post("/users/{id}/deprovision", HandleDeprovisionUser(database, hub))

        r.Get("/macros", HandleGetMacros(database))
        r.Post("/macros", HandleCreateMacro(database))
        r.Put("/macros/{id}", HandleUpdateMacro(database))
        r.Delete("/macros/{id}", HandleDeleteMacro(database))
    })

    r.Get("/api/docs/{slug}", HandleGetPublicPage(database))
    r.Post("/api/admin/login", HandleAdminLogin())

    // --- 2. Invite / Onboarding API (Unauthenticated initially, tokens checked inside handlers) ---
    r.Route("/api/invite/{token}", func(r chi.Router) {
		// Apply the database-aware middleware
		r.Use(RequireInviteToken(database))

		// These handlers now securely trust that the token is valid!
		r.Get("/", HandleGetInviteData(database))
		r.Get("/page/{slug}", HandleGetPageBySlug(database))
		r.Post("/complete", HandleCompleteOnboarding(database, hub, giteaClient))
	})

    // --- 3. Edge Agent WebSocket (Secured by Ed25519) ---
    r.Route("/api/ws", func(r chi.Router) {
		// Inject the database into the middleware
		r.Use(RequireEdgeAuth(database)) 
		
		// If the middleware passes, upgrade the connection!
		r.Get("/agent", hub.HandleWS) 
	})

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
