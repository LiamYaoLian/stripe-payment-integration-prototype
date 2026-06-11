package auth

import "context"

type userContextKey struct{}

// UserSession holds authenticated user identity from a JWT.
type UserSession struct {
	ID    string
	Email string
}

// WithUser stores the authenticated user on the context.
func WithUser(ctx context.Context, session UserSession) context.Context {
	return context.WithValue(ctx, userContextKey{}, session)
}

// UserFromContext returns the user session set by JWT middleware.
func UserFromContext(ctx context.Context) (UserSession, bool) {
	session, ok := ctx.Value(userContextKey{}).(UserSession)
	return session, ok && session.ID != "" && session.Email != ""
}
