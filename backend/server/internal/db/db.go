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
	AdminContext string `gorm:"type:text"`
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
	ExecutionLog   string `gorm:"type:text"`
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

	seedDefaultPages(database)
	seedDefaultSettings(database)
	seedVirtualAgents(database)

	log.Println("Database initialized and migrated.")
	return database
}

func seedDefaultPages(db *gorm.DB) {
	pages := []Page{
		{
			ID:      "page-invite",
			Slug:    "default-onboarding",
			Title:   "Default Account Setup",
			Content: "## Welcome {{.Username}}!\n\nYour administrator has granted you access. Please configure your permanent credentials below to activate your infrastructure access.\n\n> **Note:** SSH keys should be Ed25519 format.",
		},
		{
			ID:      "page-email",
			Slug:    "default-email",
			Title:   "Default Welcome Email",
			Content: "Hello {{.Username}},\n\nYou have been invited to the infrastructure portal. Click the link below to configure your access:\n\n{{.InviteURL}}\n\n*This link expires in 48 hours.*",
		},
		{
			ID:      "page-guide",
			Slug:    "ssh-quickstart",
			Title:   "SSH Quickstart Guide",
			Content: "## Generating an SSH Key\n\nIf you do not have an Ed25519 key, generate one by opening your terminal and running:\n\n```bash\nssh-keygen -t ed25519 -C \"{{.Email}}\"\n```\n\nCopy the contents of the `.pub` file and paste it into the onboarding form.",
		},
	}

	for _, p := range pages {
		// Use FirstOrCreate to ensure we don't overwrite user edits on server reboot
		db.Where("slug = ?", p.Slug).FirstOrCreate(&p)
	}
}

func seedDefaultSettings(db *gorm.DB) {
	settings := []SystemSetting{
		{Key: "theme", Value: "corporate"},
		{Key: "default_invite_slug", Value: "default-onboarding"},
		{Key: "default_email_slug", Value: "default-email"}, // NEW
		{Key: "user_expire_days", Value: "0"},
		{Key: "gitea_url", Value: ""},
	}
	for _, s := range settings {
		db.Where("key = ?", s.Key).FirstOrCreate(&s)
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