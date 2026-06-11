package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
	"github.com/rs/xid"
)

func TestAuthCreateGuestSession(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	accessToken, hash, err := auth.GenerateOrderAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	email := "buyer@example.com"
	orderID := xid.New().String()
	store.Orders[orderID] = &db.Order{
		ID: orderID, CustomerEmail: &email, AccessTokenHash: &hash,
	}

	h := NewAuthHandler(service.NewAuthService(store, "test-jwt-secret"))
	body, _ := json.Marshal(map[string]string{
		"email": email, "orderId": orderID, "accessToken": accessToken,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.CreateSession(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Token string `json:"token"`
			Role  string `json:"role"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Token == "" || env.Data.Role != "guest" {
		t.Fatalf("data %+v", env.Data)
	}
}
