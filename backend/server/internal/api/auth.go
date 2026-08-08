package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret should be set in ENV. Fallback for development.
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("nexus-super-secret-dev-key")
	}
	return []byte(secret)
}

type LoginRequest struct {
	Password string `json:"password"`
}

func HandleAdminLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		adminPass := os.Getenv("ADMIN_PASSWORD")
		if adminPass == "" {
			http.Error(w, "InvalidConfig: No Password Set", http.StatusInternalServerError)
			return // Fallback for local testing
		}

		if req.Password != adminPass {
			http.Error(w, "Unauthorized: Invalid password", http.StatusUnauthorized)
			return
		}

		// Generate JWT Token valid for 24 hours
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"role": "admin",
			"exp":  time.Now().Add(24 * time.Hour).Unix(),
		})

		tokenString, err := token.SignedString(getJWTSecret())
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": tokenString,
		})
	}
}