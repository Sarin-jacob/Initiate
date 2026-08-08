package db

import (
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User represents the central identity
type User struct {
	ID           string `gorm:"primaryKey;type:uuid"`
	Username     string `gorm:"uniqueIndex;not null"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string
	SSHPublicKey string
	Status       string `gorm:"default:'PENDING'"` // PENDING, ACTIVE, DISABLED, ARCHIVED
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TargetServer represents an Edge Agent
type TargetServer struct {
	ID           			string    `gorm:"primaryKey;type:uuid"`
	Name         			string    `gorm:"not null"`
	PublicKey    			string    `gorm:"uniqueIndex;not null"`
	Status       			string    `gorm:"default:'OFFLINE'"`
	Capabilities 			string    `gorm:"type:text"`
	ProvisionMacroID       	string `gorm:"type:uuid"`
	SoftDeprovisionMacroID 	string `gorm:"type:uuid"`
	HardDeprovisionMacroID 	string `gorm:"type:uuid"`
	CreatedAt    			time.Time
	UpdatedAt    			time.Time
}

type Page struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	Slug      string    `gorm:"uniqueIndex;not null"` // e.g., 'index', 'ssh-guide'
	Title     string    `gorm:"not null"`
	Content   string    `gorm:"type:text"` // The raw markdown with Go variables
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserAccess tracks what edge modules the user has
type UserAccess struct {
	ID             string `gorm:"primaryKey;type:uuid"`
	UserID         string `gorm:"index;not null"`
	TargetType     string `gorm:"not null"` // "GITEA" or "SERVER"
	TargetID       string `gorm:"not null"` // Gitea identifier or Edge Server UUID
	Status         string `gorm:"default:'PENDING'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Invitation tracks pending user invites
type Invitation struct {
	ID               string    `gorm:"primaryKey;type:uuid"`
	UserID           string    `gorm:"index;not null"`
	TokenHash        string    `gorm:"uniqueIndex;not null"`
	MarkdownTemplate string    `gorm:"type:text"`
	ExpiresAt        time.Time
	UsedAt           *time.Time
}

type Macro struct {
	ID          string    `gorm:"primaryKey;type:uuid"`
	Name        string    `gorm:"uniqueIndex;not null"` // e.g., "Standard Linux Dev"
	Description string    `gorm:"type:text"`
	Steps       string    `gorm:"type:text;not null"`   // JSON array of MacroStep
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SystemSetting struct {
	Key   string `gorm:"primaryKey"`
	Value string `gorm:"type:text"`
}

func InitDB(dsn string) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schemas
	err = database.AutoMigrate(
		&User{},
		&TargetServer{},
		&UserAccess{},
		&Invitation{},
		&Page{},
		&SystemSetting{},
		&Macro{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	seedDefaultSettings(database)
	seedVirtualAgents(database)

	log.Println("Database initialized and migrated.")
	return database
}

func seedDefaultSettings(db *gorm.DB) {
	defaults := map[string]string{
		"theme":               "corporate",
		"gitea_url":           "http://localhost:3000",
		"default_invite_slug": "index", // Which markdown page to load first
		"user_expire_days":    "0",       // 0 means never expire
	}

	for key, val := range defaults {
		var setting SystemSetting
		// Only create if it doesn't exist
		if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
			db.Create(&SystemSetting{Key: key, Value: val})
		}
	}
}

// seedVirtualAgents ensures internal systems are registered as targets with their capabilities
func seedVirtualAgents(db *gorm.DB) {
	// Gitea's internal capabilities manifest
	capabilities := `{"gitea_user":["create","delete","suspend"]}`
	
	var agent TargetServer
	if err := db.Where("id = ?", "internal-gitea").First(&agent).Error; err != nil {
		// Does not exist, create it
		db.Create(&TargetServer{
			ID:           "internal-gitea",
			Name:         "Central Gitea Server",
			PublicKey:    "internal-virtual-agent",
			Status:       "ONLINE", // Always online
			Capabilities: capabilities,
		})
	} else {
		// Update capabilities in case we added new features in the code
		db.Model(&agent).Updates(map[string]interface{}{
			"status":       "ONLINE",
			"capabilities": capabilities,
		})
	}
}