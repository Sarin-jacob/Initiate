package main

import (
    "log"
    "net/http"
    "os"

    "github.com/Sarin-jacob/Initiate/internal/agenthub"
    "github.com/Sarin-jacob/Initiate/internal/api"
    "github.com/Sarin-jacob/Initiate/internal/db"
    "github.com/Sarin-jacob/Initiate/internal/gitea"
    "github.com/Sarin-jacob/Initiate/internal/mailer"
)

func main() {
    log.Println("Starting EdgeAuth Central Controller...")

    // 1. Initialize SQLite Database
    database := db.InitDB("edgeauth.db")

    
    giteaClient := gitea.NewClient(
        os.Getenv("GITEA_INTERNAL_URL"),
        os.Getenv("GITEA_ADMIN_TOKEN"),
    )
    
    // 2. Initialize Subsystems
    hub := agenthub.NewHub(database, giteaClient)

    emailConfig := mailer.SMTPConfig{
        Host:     os.Getenv("SMTP_HOST"),
        Port:     os.Getenv("SMTP_PORT"),
        Username: os.Getenv("SMTP_USER"),
        Password: os.Getenv("SMTP_PASS"),
        From:     os.Getenv("SMTP_FROM"), // e.g., "admin@yourdomain.com"
    }
    emailService := mailer.NewMailer(emailConfig)
    
    baseURL := os.Getenv("BASE_URL")

    // 3. Initialize Router with Injected Dependencies
    router := api.NewRouter(database, hub, giteaClient, emailService, baseURL)

    // 4. Start HTTP Server
    port := ":8080"
    log.Printf("Server listening on %s\n", port)
    if err := http.ListenAndServe(port, router); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}