package internal

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

// LoadPrivateKey reads the raw Ed25519 private key from disk
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid ed25519 private key size")
	}
	return keyBytes, nil
}

// SignChallenge signs the random nonce sent by the Central Server
func SignChallenge(privateKey ed25519.PrivateKey, nonce string) string {
	signature := ed25519.Sign(privateKey, []byte(nonce))
	return hex.EncodeToString(signature)
}

// GetPinnedTLSConfig builds a TLS config that strictly pins the server certificate hash
func GetPinnedTLSConfig(expectedPin string) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // We disable standard CA verification...
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// ...and implement our own perfect hash matching (Certificate Pinning)
			if len(rawCerts) == 0 {
				return errors.New("no certificates provided by server")
			}
			
			hash := sha256.Sum256(rawCerts[0])
			hashStr := "sha256:" + hex.EncodeToString(hash[:])
			
			if hashStr != expectedPin {
				return fmt.Errorf("FATAL: Certificate pin mismatch! Expected %s, got %s", expectedPin, hashStr)
			}
			return nil
		},
	}
}