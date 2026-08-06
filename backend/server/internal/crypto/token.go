package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateInviteToken creates a 32-byte cryptographically secure random token.
// It returns the raw URL-safe token (to email the user) and the SHA-256 hash (to store in DB).
func GenerateInviteToken() (rawToken string, tokenHash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	
	// Raw token for the email link (URL Safe)
	rawToken = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	
	// Hash for the database
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(hash[:])
	
	return rawToken, tokenHash, nil
}

// HashToken is a helper to hash the incoming token from the URL for DB lookup
func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}