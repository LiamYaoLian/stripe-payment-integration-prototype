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

const (
	userSessionTTL    = 7 * 24 * time.Hour
	minPasswordLength = 8
)

// UserSessionResult is returned after register or login.
type UserSessionResult struct {
	Token     string
	ExpiresAt time.Time
	User      UserProfile
}

// UserProfile is the public user identity.
type UserProfile struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type authCustomerStore interface {
	CreateCustomer(ctx context.Context, id, email, passwordHash string) (*db.Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*db.Customer, error)
	GetCustomerByID(ctx context.Context, id string) (*db.Customer, error)
	LinkOrdersToCustomer(ctx context.Context, customerID, email string) error
}

// AuthService handles user registration and login.
type AuthService struct {
	store     authCustomerStore
	jwtSecret string
}

func NewAuthService(store authCustomerStore, jwtSecret string) *AuthService {
	return &AuthService{store: store, jwtSecret: jwtSecret}
}

// Register creates a new account and issues a user JWT.
func (s *AuthService) Register(ctx context.Context, email, password string) (*UserSessionResult, error) {
	email, password, err := validateCredentials(email, password)
	if err != nil {
		return nil, err
	}
	if s.jwtSecret == "" {
		return nil, &api.AppError{Status: 503, Code: "AUTH_DISABLED", Message: "auth not configured"}
	}

	existing, err := s.store.GetCustomerByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, &api.AppError{Status: 409, Code: "EMAIL_TAKEN", Message: "email already registered"}
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	customer, err := s.store.CreateCustomer(ctx, xid.New().String(), email, hash)
	if err != nil {
		return nil, err
	}
	if err := s.store.LinkOrdersToCustomer(ctx, customer.ID, customer.Email); err != nil {
		return nil, err
	}

	return s.issueSession(customer)
}

// Login verifies credentials and issues a user JWT.
func (s *AuthService) Login(ctx context.Context, email, password string) (*UserSessionResult, error) {
	email, password, err := validateCredentials(email, password)
	if err != nil {
		return nil, err
	}
	if s.jwtSecret == "" {
		return nil, &api.AppError{Status: 503, Code: "AUTH_DISABLED", Message: "auth not configured"}
	}

	customer, err := s.store.GetCustomerByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if customer == nil || !auth.CheckPassword(customer.PasswordHash, password) {
		return nil, invalidCredentials()
	}
	if err := s.store.LinkOrdersToCustomer(ctx, customer.ID, customer.Email); err != nil {
		return nil, err
	}

	return s.issueSession(customer)
}

// GetUser returns the profile for an authenticated customer ID.
func (s *AuthService) GetUser(ctx context.Context, customerID string) (*UserProfile, error) {
	if customerID == "" {
		return nil, &api.AppError{Status: 401, Code: "UNAUTHORIZED", Message: "user session required"}
	}
	customer, err := s.store.GetCustomerByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, &api.AppError{Status: 401, Code: "UNAUTHORIZED", Message: "user not found"}
	}
	return &UserProfile{ID: customer.ID, Email: customer.Email}, nil
}

func (s *AuthService) issueSession(customer *db.Customer) (*UserSessionResult, error) {
	token, expiresAt, err := auth.IssueUserToken(s.jwtSecret, customer.ID, customer.Email, userSessionTTL)
	if err != nil {
		return nil, err
	}
	return &UserSessionResult{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      UserProfile{ID: customer.ID, Email: customer.Email},
	}, nil
}

func validateCredentials(email, password string) (string, string, error) {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return "", "", &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email and password required"}
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", "", &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email invalid"}
	}
	if len(password) < minPasswordLength {
		return "", "", &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "password must be at least 8 characters"}
	}
	return email, password, nil
}

func invalidCredentials() *api.AppError {
	return &api.AppError{Status: 401, Code: "INVALID_CREDENTIALS", Message: "invalid email or password"}
}
