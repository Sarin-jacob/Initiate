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
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TargetServer represents an Edge Agent
type TargetServer struct {
	ID           string `gorm:"primaryKey;type:uuid"`
	Name         string `gorm:"uniqueIndex;not null"`
	PublicKey    string `gorm:"not null"`
	Status       string `gorm:"default:'OFFLINE'"`
	Capabilities string `gorm:"type:text"`
	LastSeen     time.Time
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
	UserID     string `gorm:"primaryKey"`
	TargetType string `gorm:"primaryKey"` // 'GITEA' or 'SERVER'
	TargetID   string `gorm:"primaryKey"` // empty if GITEA, TargetServer.ID if SERVER
	Status     string `gorm:"default:'PENDING'"`
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
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database initialized and migrated.")
	return database
}