package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Sarin-jacob/Initiate/internal/crypto"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// RequireAdmin verifies the Admin JWT
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized: Missing or invalid token", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized: Token expired or invalid", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireEdgeAuth secures the WebSocket connection for Edge Agents
func RequireEdgeAuth(database *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The Edge Agent ONLY sends its Public Key
			agentPubKey := r.Header.Get("X-Agent-Pubkey")

			if agentPubKey == "" {
				http.Error(w, "Unauthorized: Missing Agent Public Key", http.StatusUnauthorized)
				return
			}

			// Look up the agent by its Public Key
			var server db.TargetServer
			if err := database.Where("public_key = ?", agentPubKey).First(&server).Error; err != nil {
				http.Error(w, "Forbidden: Unregistered Public Key", http.StatusForbidden)
				return
			}

			// Attach the verified Agent ID to the context so the WebSocket Hub knows who it is
			ctx := context.WithValue(r.Context(), "edge_agent_id", server.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireInviteToken validates the short-lived invitation token
func RequireInviteToken(database *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := chi.URLParam(r, "token")
			if rawToken == "" {
				http.Error(w, "Unauthorized: Missing invite token", http.StatusUnauthorized)
				return
			}

			tokenHash := crypto.HashToken(rawToken)

			var invite db.Invitation
			if err := database.Where("token_hash = ?", tokenHash).First(&invite).Error; err != nil {
				http.Error(w, "Unauthorized: Invalid invite token", http.StatusUnauthorized)
				return
			}

			// Reject if expired OR already used
			if invite.UsedAt != nil || time.Now().After(invite.ExpiresAt) {
				http.Error(w, "Unauthorized: Token expired or already used", http.StatusForbidden)
				return
			}

			// Pass the ENTIRE invite struct down the chain so handlers don't have to query the DB!
			ctx := context.WithValue(r.Context(), "invite", invite)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}