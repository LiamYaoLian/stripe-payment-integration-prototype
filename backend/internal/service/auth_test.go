package service

import (
	"context"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
	"github.com/rs/xid"
)

func newTestAuthService() (*AuthService, *testutil.FakeAuthStore) {
	store := testutil.NewFakeAuthStore()
	return NewAuthService(store, store, store, "http://localhost:5173", "development"), store
}

func TestRegisterRequiresValidInput(t *testing.T) {
	svc, _ := newTestAuthService()

	_, err := svc.Register(context.Background(), "", "password123")
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 400 {
		t.Fatalf("expected validation error, got %v", err)
	}

	_, err = svc.Register(context.Background(), "buyer@example.com", "short")
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 400 {
		t.Fatalf("expected password validation error, got %v", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	svc, _ := newTestAuthService()

	_, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Register(context.Background(), "buyer@example.com", "otherpass1")
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 409 || apiErr.Code != "EMAIL_TAKEN" {
		t.Fatalf("expected email taken, got %v", err)
	}
}

func TestRegisterSuccess(t *testing.T) {
	svc, store := newTestAuthService()

	result, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionToken == "" || result.User.Email != "buyer@example.com" {
		t.Fatalf("result %+v", result)
	}
	if len(store.EmailVerifyByTokenHash) != 1 {
		t.Fatal("expected email verification token")
	}

	session, err := store.GetUserSessionByTokenHash(context.Background(), auth.HashOpaqueToken(result.SessionToken))
	if err != nil || session == nil {
		t.Fatalf("session %+v err=%v", session, err)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	svc, _ := newTestAuthService()

	_, err := svc.Login(context.Background(), "missing@example.com", "password123")
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 401 {
		t.Fatalf("expected unauthorized, got %v", err)
	}

	_, err = svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Login(context.Background(), "buyer@example.com", "wrongpass1")
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 401 {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestLoginSuccess(t *testing.T) {
	svc, _ := newTestAuthService()

	_, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.Login(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionToken == "" {
		t.Fatal("expected session token")
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	svc, store := newTestAuthService()
	result, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(context.Background(), result.SessionToken); err != nil {
		t.Fatal(err)
	}
	if len(store.SessionsByTokenHash) != 0 {
		t.Fatal("expected session revoked")
	}
}

func TestGetUser(t *testing.T) {
	svc, _ := newTestAuthService()

	result, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	user, err := svc.GetUser(context.Background(), result.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "buyer@example.com" {
		t.Fatalf("user %+v", user)
	}

	_, err = svc.GetUser(context.Background(), xid.New().String())
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 401 {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	svc, store := newTestAuthService()
	result, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	resetToken := "reset-token-123"
	store.StorePasswordResetToken(resetToken, result.User.ID)

	if err := svc.ResetPassword(context.Background(), resetToken, "newpassword1"); err != nil {
		t.Fatal(err)
	}

	_, err = svc.Login(context.Background(), "buyer@example.com", "newpassword1")
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
}

func TestVerifyEmail(t *testing.T) {
	svc, store := newTestAuthService()
	result, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if result.User.EmailVerified {
		t.Fatal("expected unverified email")
	}

	verifyToken := "verify-token"
	store.EmailVerifyByTokenHash[auth.HashOpaqueToken(verifyToken)] = result.User.ID

	if err := svc.VerifyEmail(context.Background(), verifyToken); err != nil {
		t.Fatal(err)
	}

	user, err := svc.GetUser(context.Background(), result.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !user.EmailVerified {
		t.Fatal("expected verified email")
	}
}
