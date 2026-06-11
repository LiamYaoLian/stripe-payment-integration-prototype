package service

import (
	"context"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
	"github.com/rs/xid"
)

func TestCreateGuestSessionRequiresOrderProof(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	svc := NewAuthService(store, "test-secret")

	_, err := svc.CreateGuestSession(context.Background(), "buyer@example.com", "", "")
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 400 {
		t.Fatalf("expected validation error, got %v", err)
	}

	orderID := xid.New().String()
	_, err = svc.CreateGuestSession(context.Background(), "buyer@example.com", orderID, "token")
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 401 {
		t.Fatalf("expected unauthorized for missing order, got %v", err)
	}
}

func TestCreateGuestSessionRejectsEmailMismatch(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	token, hash, err := auth.GenerateOrderAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	email := "buyer@example.com"
	orderID := xid.New().String()
	store.Orders[orderID] = &db.Order{
		ID: orderID, CustomerEmail: &email, AccessTokenHash: &hash,
	}
	svc := NewAuthService(store, "test-secret")

	_, err = svc.CreateGuestSession(context.Background(), "other@example.com", orderID, token)
	if apiErr, ok := err.(*api.AppError); !ok || apiErr.Status != 401 {
		t.Fatalf("expected unauthorized for email mismatch, got %v", err)
	}
}

func TestCreateGuestSessionSuccess(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	token, hash, err := auth.GenerateOrderAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	email := "buyer@example.com"
	orderID := xid.New().String()
	store.Orders[orderID] = &db.Order{
		ID: orderID, CustomerEmail: &email, AccessTokenHash: &hash,
	}
	svc := NewAuthService(store, "test-secret")

	result, err := svc.CreateGuestSession(context.Background(), email, orderID, token)
	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" || result.Role != auth.RoleGuest {
		t.Fatalf("result %+v", result)
	}
	claims, err := auth.VerifyGuestToken("test-secret", result.Token)
	if err != nil || claims.Email != email {
		t.Fatalf("claims %+v err=%v", claims, err)
	}
}
