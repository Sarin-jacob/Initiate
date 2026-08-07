package api

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/db"
)

// UserWithAccess bundles the core user profile with their target provisions
type UserWithAccess struct {
	db.User
	AccessList []db.UserAccess `json:"access_list"`
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