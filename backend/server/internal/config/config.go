package config

import (
	"log"
	"os"
	"strconv"
)

// AppConfig holds all environment variables in a strongly-typed struct
type AppConfig struct {
	Port             int
	Host			 string
	BaseURL          string
	DBPath           string
	AdminPassword    string
	JWTSecret        string
	StaticFolder	 string
	
	GiteaInternalURL string
	GiteaAdminToken  string
	GiteaExternalURL string
	
	SMTPHost         string
	SMTPPort         string
	SMTPUser         string
	SMTPPass         string
	SMTPFrom         string
}

// App holds the global configuration state after Load() is called
var App *AppConfig

// Load reads all environment variables, applies defaults, and populates the App instance
func Load() {
	App = &AppConfig{
		Port:             getEnvAsInt("PORT", 8080),
		Host: 			  getEnv("HOST","0.0.0.0"),	
		BaseURL:          getEnv("BASE_URL", "http://localhost:8080"),
		DBPath:           getEnv("DB_PATH", "./nexus.db"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "changeme123"),
		JWTSecret:        getEnv("JWT_SECRET", "default_insecure_secret_key"),
		StaticFolder:     getEnv("STATIC_FOLDER","static"),

		GiteaInternalURL: getEnv("GITEA_INTERNAL_URL", ""),
		GiteaExternalURL: getEnv("GITEA_EXTERNAL_URL", ""),
		GiteaAdminToken:  getEnv("GITEA_ADMIN_TOKEN", ""),
		
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         getEnv("SMTP_PORT", "587"),
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPass:         getEnv("SMTP_PASS", ""),
		SMTPFrom:         getEnv("SMTP_FROM", "Nexus Admin <admin@localhost>"),
	}

	// Optional: Log a warning if running with default security keys
	if App.JWTSecret == "default_insecure_secret_key" {
		log.Println("WARNING: JWT_SECRET is using the default insecure fallback. Please set it in your environment.")
	}
}

// Helper to read string with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// Helper to read int with fallback
func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}