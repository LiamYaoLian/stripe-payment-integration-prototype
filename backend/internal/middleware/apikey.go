package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
)

// RequireAPIKey protects internal endpoints with a shared secret header.
func RequireAPIKey(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expectedKey == "" {
				api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "endpoint disabled")
				return
			}
			provided := r.Header.Get("X-API-Key")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(expectedKey)) != 1 {
				api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid api key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
