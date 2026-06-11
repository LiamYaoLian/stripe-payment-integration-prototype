package middleware

import (
	"net/http"
	"strings"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
)

// RequireUserJWT validates Bearer user session tokens.
func RequireUserJWT(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
				return
			}
			claims, err := auth.VerifyUserToken(secret, strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}
			session := auth.UserSession{ID: claims.Subject, Email: claims.Email}
			next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), session)))
		})
	}
}
