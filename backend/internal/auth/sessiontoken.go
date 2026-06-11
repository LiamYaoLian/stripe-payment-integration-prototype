package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateOpaqueToken returns a URL-safe random token and its SHA-256 hex hash.
func GenerateOpaqueToken() (token string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashOpaqueToken(token), nil
}

// HashOpaqueToken hashes a token for database storage.
func HashOpaqueToken(token string) string {
	return HashOrderAccessToken(token)
}

// VerifyOpaqueToken compares a presented token against a stored hash.
func VerifyOpaqueToken(token, storedHash string) bool {
	return VerifyOrderAccessToken(token, storedHash)
}
