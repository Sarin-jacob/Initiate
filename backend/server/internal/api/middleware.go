package api

import (
	"context"
	"net/http"
	"strings"
)

// RequireAdmin verifies the Admin JWT or Secure Cookie
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized: Missing or invalid token", http.StatusUnauthorized)
			return
		}
		
		// TODO: Parse JWT and validate admin signature here
		
		next.ServeHTTP(w, r)
	})
}

// RequireEdgeAuth handles the Ed25519 Challenge-Response for WebSockets
func RequireEdgeAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentPubKey := r.Header.Get("X-Agent-Pubkey")
		if agentPubKey == "" {
			http.Error(w, "Unauthorized: Missing Agent Public Key", http.StatusUnauthorized)
			return
		}

		// TODO: Look up agentPubKey in DB. 
		// If found, initiate WebSocket upgrade and issue cryptographic challenge.
		// For the HTTP middleware phase, we just ensure headers exist.

		next.ServeHTTP(w, r)
	})
}

// RequireInviteToken validates the short-lived invitation token
func RequireInviteToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Token is usually in URL path or query params
		// TODO: Validate token hash against DB and check expiration

		// If valid, attach UserID to context for the handler to use
		ctx := context.WithValue(r.Context(), "invite_user_id", "mock-uuid")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}