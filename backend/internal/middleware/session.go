package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

const sessionSlidingTTL = 7 * 24 * time.Hour

// SessionLookup resolves and optionally extends authenticated sessions.
type SessionLookup interface {
	GetUserSessionByTokenHash(ctx context.Context, tokenHash string) (*db.UserSession, error)
	TouchUserSession(ctx context.Context, sessionID string, expiresAt time.Time) error
}

// RequireUserSession validates the httpOnly session cookie against Postgres.
func RequireUserSession(store SessionLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := auth.SessionTokenFromRequest(r)
			if token == "" {
				api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
				return
			}
			session, err := store.GetUserSessionByTokenHash(r.Context(), auth.HashOpaqueToken(token))
			if err != nil {
				api.WriteError(w, http.StatusServiceUnavailable, "SESSION_UNAVAILABLE", "session check failed")
				return
			}
			if session == nil {
				api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired session")
				return
			}
			newExpiry := time.Now().UTC().Add(sessionSlidingTTL)
			_ = store.TouchUserSession(r.Context(), session.ID, newExpiry)
			auth.SetSessionCookie(w, token, newExpiry, r.TLS != nil)

			user := auth.UserSession{ID: session.CustomerID, Email: session.Email}
			next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), user)))
		})
	}
}
