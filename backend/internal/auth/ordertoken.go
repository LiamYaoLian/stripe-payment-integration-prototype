package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

// GenerateOrderAccessToken returns a client token and its SHA-256 hex hash for storage.
func GenerateOrderAccessToken() (token string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashOrderAccessToken(token), nil
}

// HashOrderAccessToken hashes a token for database storage.
func HashOrderAccessToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// VerifyOrderAccessToken compares a presented token against a stored hash.
func VerifyOrderAccessToken(token, storedHash string) bool {
	if token == "" || storedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(HashOrderAccessToken(token)), []byte(storedHash)) == 1
}
