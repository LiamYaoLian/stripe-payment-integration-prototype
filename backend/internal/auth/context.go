package auth

import "context"

type guestEmailKey struct{}

// WithGuestEmail stores the authenticated guest email on the context.
func WithGuestEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, guestEmailKey{}, email)
}

// GuestEmailFromContext returns the guest email set by JWT middleware.
func GuestEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(guestEmailKey{}).(string)
	return email, ok && email != ""
}
