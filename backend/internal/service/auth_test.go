package service

import (
	"context"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
	"github.com/rs/xid"
)

func TestRegisterRequiresValidInput(t *testing.T) {
	store := testutil.NewFakeCustomerStore()
	svc := NewAuthService(store, "test-secret")

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
	store := testutil.NewFakeCustomerStore()
	svc := NewAuthService(store, "test-secret")

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
	store := testutil.NewFakeCustomerStore()
	email := "buyer@example.com"
	svc := NewAuthService(store, "test-secret")

	result, err := svc.Register(context.Background(), email, "password123")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" || result.User.Email != email {
		t.Fatalf("result %+v", result)
	}

	claims, err := auth.VerifyUserToken("test-secret", result.Token)
	if err != nil || claims.Email != email {
		t.Fatalf("claims %+v err=%v", claims, err)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	store := testutil.NewFakeCustomerStore()
	svc := NewAuthService(store, "test-secret")

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
	store := testutil.NewFakeCustomerStore()
	svc := NewAuthService(store, "test-secret")

	_, err := svc.Register(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.Login(context.Background(), "buyer@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" {
		t.Fatal("expected token")
	}
}

func TestGetUser(t *testing.T) {
	store := testutil.NewFakeCustomerStore()
	svc := NewAuthService(store, "test-secret")

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
