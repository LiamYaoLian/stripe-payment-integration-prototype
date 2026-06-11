package service

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/rs/xid"
)

const guestSessionTTL = 24 * time.Hour

// GuestSessionResult is returned after creating a guest auth session.
type GuestSessionResult struct {
	Token     string
	ExpiresAt time.Time
	Role      string
}

type authOrderStore interface {
	GetOrderByID(ctx context.Context, id string) (*db.Order, error)
}

// AuthService issues guest JWT sessions after email ownership is proven via an order access token.
type AuthService struct {
	store     authOrderStore
	jwtSecret string
}

func NewAuthService(store authOrderStore, jwtSecret string) *AuthService {
	return &AuthService{store: store, jwtSecret: jwtSecret}
}

// CreateGuestSession validates email, proves ownership with orderId + accessToken, then issues a guest JWT.
func (s *AuthService) CreateGuestSession(ctx context.Context, email, orderID, accessToken string) (*GuestSessionResult, error) {
	email = strings.TrimSpace(email)
	orderID = strings.TrimSpace(orderID)
	accessToken = strings.TrimSpace(accessToken)

	if email == "" || orderID == "" || accessToken == "" {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email, orderId, and accessToken required"}
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email invalid"}
	}
	if s.jwtSecret == "" {
		return nil, &api.AppError{Status: 503, Code: "AUTH_DISABLED", Message: "guest auth not configured"}
	}
	if !validGuestProofOrderID(orderID) {
		return nil, guestAuthProofFailed()
	}

	order, err := s.store.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !verifyGuestEmailProof(order, email, accessToken) {
		return nil, guestAuthProofFailed()
	}

	token, expiresAt, err := auth.IssueGuestToken(s.jwtSecret, email, guestSessionTTL)
	if err != nil {
		return nil, err
	}
	return &GuestSessionResult{Token: token, ExpiresAt: expiresAt, Role: auth.RoleGuest}, nil
}

func validGuestProofOrderID(id string) bool {
	if len(id) != 20 {
		return false
	}
	_, err := xid.FromString(id)
	return err == nil
}

func verifyGuestEmailProof(order *db.Order, email, accessToken string) bool {
	if order == nil || order.CustomerEmail == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(*order.CustomerEmail)) {
		return false
	}
	hash := ""
	if order.AccessTokenHash != nil {
		hash = *order.AccessTokenHash
	}
	return auth.VerifyOrderAccessToken(accessToken, hash)
}

func guestAuthProofFailed() *api.AppError {
	return &api.AppError{Status: 401, Code: "UNAUTHORIZED", Message: "email ownership could not be verified"}
}
