package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/api"
	"github.com/Sarin-jacob/Initiate/internal/config"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/gitea"
	"github.com/Sarin-jacob/Initiate/internal/mailer"
	"github.com/Sarin-jacob/Initiate/internal/workers"
)

func main() {
    log.Println("Starting Nexus Central Controller...")

    // 1. Initialize SQLite Database
    config.Load()
    database := db.InitDB("nexus.db")

    
    giteaClient := gitea.NewClient(
        config.App.GiteaInternalURL,
        config.App.GiteaAdminToken,
    )
    
    // 2. Initialize Subsystems
    hub := agenthub.NewHub(database, giteaClient)

    emailConfig := mailer.SMTPConfig{
        Host:    config.App.SMTPHost,
        Port:     config.App.SMTPPort,
        Username: config.App.SMTPUser,
        Password: config.App.SMTPPass,
        From:     config.App.SMTPFrom, // e.g., "admin@yourdomain.com"
    }
    emailService := mailer.NewMailer(emailConfig)
    
    baseURL := config.App.BaseURL

    // 3. Initialize Router with Injected Dependencies
    router := api.NewRouter(database, hub, giteaClient, emailService, baseURL)

    workers.StartExpirationCron(database, hub)

    // 4. Start HTTP Server
    port := fmt.Sprintf("%v:%v",config.App.Host,config.App.Port)
    log.Printf("Server listening on %s\n", port)
    if err := http.ListenAndServe(port, router); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}