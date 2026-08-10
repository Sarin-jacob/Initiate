package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/go-chi/chi/v5"
)

// UserWithAccess bundles the core user profile with their target provisions
type UserWithAccess struct {
	db.User
	AccessList []db.UserAccess `json:"access_list"`
}

type UpdateExpiryRequest struct {
	ExpireAmount int    `json:"expire_amount"`
	ExpireUnit   string `json:"expire_unit"`
}


// HandleListUsers returns all users and their associated module access states
func HandleListUsers(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var users []db.User
		if err := database.Find(&users).Error; err != nil {
			http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
			return
		}

		var accesses []db.UserAccess
		if err := database.Find(&accesses).Error; err != nil {
			http.Error(w, "Failed to fetch user access matrix", http.StatusInternalServerError)
			return
		}

		// Map accesses to their respective User ID
		accessMap := make(map[string][]db.UserAccess)
		for _, acc := range accesses {
			accessMap[acc.UserID] = append(accessMap[acc.UserID], acc)
		}

		// Build the final response array
		var result []UserWithAccess
		for _, u := range users {
			// Sanitize password hash before sending to frontend
			u.PasswordHash = "" 
			
			result = append(result, UserWithAccess{
				User:       u,
				AccessList: accessMap[u.ID],
			})
		}

		// Ensure we send an empty JSON array `[]` instead of `null` if DB is empty
		if result == nil {
			result = []UserWithAccess{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// HandleUpdateUserExpiration changes the ExpiresAt date for an active user
func HandleUpdateUserExpiration(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "id")
		
		var req UpdateExpiryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		var expiresAt *time.Time
		if req.ExpireAmount > 0 {
			expTime := time.Now()
			switch req.ExpireUnit {
			case "days":
				expTime = expTime.AddDate(0, 0, req.ExpireAmount)
			case "weeks":
				expTime = expTime.AddDate(0, 0, req.ExpireAmount*7)
			case "months":
				expTime = expTime.AddDate(0, req.ExpireAmount, 0)
			case "years":
				expTime = expTime.AddDate(req.ExpireAmount, 0, 0)
			default:
				expTime = expTime.AddDate(0, 0, req.ExpireAmount)
			}
			expiresAt = &expTime
		}

		// Update the database record
		if err := database.Model(&db.User{}).Where("id = ?", userID).Update("expires_at", expiresAt).Error; err != nil {
			http.Error(w, "Failed to update expiration", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func HandleForceRemoveUser(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "id")

		tx := database.Begin()
		
		// 1. Remove associated user access rows
		if err := tx.Where("user_id = ?", userID).Delete(&db.UserAccess{}).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to remove user access records", http.StatusInternalServerError)
			return
		}

		// 2. Remove associated invitations
		if err := tx.Where("user_id = ?", userID).Delete(&db.Invitation{}).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to remove user invitations", http.StatusInternalServerError)
			return
		}

		// 3. Delete the user record itself
		if err := tx.Where("id = ?", userID).Delete(&db.User{}).Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to delete user record", http.StatusInternalServerError)
			return
		}

		tx.Commit()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "User forcefully purged from database."})
	}
}