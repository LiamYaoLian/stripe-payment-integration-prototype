package service

import (
	"context"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/rs/xid"
)

const (
	userSessionTTL         = 7 * 24 * time.Hour
	passwordResetTTL       = time.Hour
	emailVerificationTTL   = 24 * time.Hour
	minPasswordLength      = 8
)

// UserSessionResult is returned after register or login.
type UserSessionResult struct {
	SessionToken string
	ExpiresAt    time.Time
	User         UserProfile
}

// UserProfile is the public user identity.
type UserProfile struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
}

type authCustomerStore interface {
	CreateCustomer(ctx context.Context, id, email, passwordHash string) (*db.Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*db.Customer, error)
	GetCustomerByID(ctx context.Context, id string) (*db.Customer, error)
	UpdateCustomerPassword(ctx context.Context, customerID, passwordHash string) error
	MarkCustomerEmailVerified(ctx context.Context, customerID string) error
}

type authSessionStore interface {
	CreateUserSession(ctx context.Context, id, tokenHash, customerID string, expiresAt time.Time) error
	GetUserSessionByTokenHash(ctx context.Context, tokenHash string) (*db.UserSession, error)
	RevokeUserSession(ctx context.Context, sessionID string) error
	RevokeAllUserSessions(ctx context.Context, customerID string) error
}

type authTokenStore interface {
	CreatePasswordResetToken(ctx context.Context, id, tokenHash, customerID string, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, tokenHash string) (string, error)
	CreateEmailVerificationToken(ctx context.Context, id, tokenHash, customerID string, expiresAt time.Time) error
	ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (string, error)
}

// AuthService handles user registration, login, and account recovery.
type AuthService struct {
	customers   authCustomerStore
	sessions    authSessionStore
	tokens      authTokenStore
	frontendURL string
	secureLogs  bool
}

func NewAuthService(customers authCustomerStore, sessions authSessionStore, tokens authTokenStore, frontendURL string, env string) *AuthService {
	return &AuthService{
		customers:   customers,
		sessions:    sessions,
		tokens:      tokens,
		frontendURL: strings.TrimRight(frontendURL, "/"),
		secureLogs:  env == "production",
	}
}

// Register creates a new account and issues a server-side session.
func (s *AuthService) Register(ctx context.Context, email, password string) (*UserSessionResult, error) {
	email, password, err := validateCredentials(email, password)
	if err != nil {
		return nil, err
	}

	existing, err := s.customers.GetCustomerByEmail(ctx, email)
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

	customer, err := s.customers.CreateCustomer(ctx, xid.New().String(), email, hash)
	if err != nil {
		return nil, err
	}

	if err := s.issueEmailVerification(ctx, customer.ID); err != nil {
		return nil, err
	}

	return s.createSession(ctx, customer)
}

// Login verifies credentials and issues a server-side session.
func (s *AuthService) Login(ctx context.Context, email, password string) (*UserSessionResult, error) {
	email, password, err := validateCredentials(email, password)
	if err != nil {
		return nil, err
	}

	customer, err := s.customers.GetCustomerByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if customer == nil || !auth.CheckPassword(customer.PasswordHash, password) {
		return nil, invalidCredentials()
	}

	return s.createSession(ctx, customer)
}

// Logout revokes the active session.
func (s *AuthService) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	session, err := s.sessions.GetUserSessionByTokenHash(ctx, auth.HashOpaqueToken(sessionToken))
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}
	return s.sessions.RevokeUserSession(ctx, session.ID)
}

// GetUser returns the profile for an authenticated customer ID.
func (s *AuthService) GetUser(ctx context.Context, customerID string) (*UserProfile, error) {
	if customerID == "" {
		return nil, &api.AppError{Status: 401, Code: "UNAUTHORIZED", Message: "user session required"}
	}
	customer, err := s.customers.GetCustomerByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, &api.AppError{Status: 401, Code: "UNAUTHORIZED", Message: "user not found"}
	}
	return toUserProfile(customer), nil
}

// RequestPasswordReset creates a reset token. Always succeeds from the caller's perspective.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email required"}
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "email invalid"}
	}

	customer, err := s.customers.GetCustomerByEmail(ctx, email)
	if err != nil {
		return err
	}
	if customer == nil {
		return nil
	}

	token, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(passwordResetTTL)
	if err := s.tokens.CreatePasswordResetToken(ctx, xid.New().String(), hash, customer.ID, expiresAt); err != nil {
		return err
	}

	s.logAccountLink("password reset", s.frontendURL+"/reset-password?token="+token)
	return nil
}

// ResetPassword consumes a reset token and sets a new password.
func (s *AuthService) ResetPassword(ctx context.Context, token, password string) error {
	password = strings.TrimSpace(password)
	if token == "" || password == "" {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "token and password required"}
	}
	if len(password) < minPasswordLength {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "password must be at least 8 characters"}
	}

	customerID, err := s.tokens.ConsumePasswordResetToken(ctx, auth.HashOpaqueToken(token))
	if err != nil {
		return err
	}
	if customerID == "" {
		return &api.AppError{Status: 400, Code: "INVALID_TOKEN", Message: "invalid or expired reset token"}
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.customers.UpdateCustomerPassword(ctx, customerID, hash); err != nil {
		return err
	}
	return s.sessions.RevokeAllUserSessions(ctx, customerID)
}

// VerifyEmail marks a customer's email as verified.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "token required"}
	}

	customerID, err := s.tokens.ConsumeEmailVerificationToken(ctx, auth.HashOpaqueToken(token))
	if err != nil {
		return err
	}
	if customerID == "" {
		return &api.AppError{Status: 400, Code: "INVALID_TOKEN", Message: "invalid or expired verification token"}
	}
	return s.customers.MarkCustomerEmailVerified(ctx, customerID)
}

func (s *AuthService) createSession(ctx context.Context, customer *db.Customer) (*UserSessionResult, error) {
	token, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(userSessionTTL)
	sessionID := xid.New().String()
	if err := s.sessions.CreateUserSession(ctx, sessionID, hash, customer.ID, expiresAt); err != nil {
		return nil, err
	}
	return &UserSessionResult{
		SessionToken: token,
		ExpiresAt:    expiresAt,
		User:         *toUserProfile(customer),
	}, nil
}

func (s *AuthService) issueEmailVerification(ctx context.Context, customerID string) error {
	token, hash, err := auth.GenerateOpaqueToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(emailVerificationTTL)
	if err := s.tokens.CreateEmailVerificationToken(ctx, xid.New().String(), hash, customerID, expiresAt); err != nil {
		return err
	}
	s.logAccountLink("email verification", s.frontendURL+"/verify-email?token="+token)
	return nil
}

func (s *AuthService) logAccountLink(kind, link string) {
	if s.secureLogs {
		slog.Info(kind+" link issued", "expires_in", emailVerificationTTL.String())
		return
	}
	slog.Info(kind+" link issued", "url", link)
}

func toUserProfile(customer *db.Customer) *UserProfile {
	return &UserProfile{
		ID:            customer.ID,
		Email:         customer.Email,
		EmailVerified: customer.EmailVerifiedAt != nil,
	}
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
