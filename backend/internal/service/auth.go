package service

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
)

const guestSessionTTL = 7 * 24 * time.Hour

// GuestSessionResult is returned after creating a guest auth session.
type GuestSessionResult struct {
	Token     string
	ExpiresAt time.Time
	Role      string
}

// AuthService issues guest JWT sessions for authenticated order lookup.
type AuthService struct {
	jwtSecret string
}

func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{jwtSecret: jwtSecret}
}

// CreateGuestSession validates email and issues a guest JWT.
func (s *AuthService) CreateGuestSession(_ context.Context, email string) (*GuestSessionResult, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email required"}
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email invalid"}
	}
	if s.jwtSecret == "" {
		return nil, &api.AppError{Status: 503, Code: "AUTH_DISABLED", Message: "guest auth not configured"}
	}

	token, expiresAt, err := auth.IssueGuestToken(s.jwtSecret, email, guestSessionTTL)
	if err != nil {
		return nil, err
	}
	return &GuestSessionResult{Token: token, ExpiresAt: expiresAt, Role: auth.RoleGuest}, nil
}
