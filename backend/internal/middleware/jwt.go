package middleware

import (
	"net/http"
	"strings"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
)

// RequireGuestJWT validates Bearer guest session tokens.
func RequireGuestJWT(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
				return
			}
			claims, err := auth.VerifyGuestToken(secret, strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithGuestEmail(r.Context(), claims.Email)))
		})
	}
}
