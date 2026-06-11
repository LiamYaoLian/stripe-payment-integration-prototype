package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

func TestAuthCreateGuestSession(t *testing.T) {
	h := NewAuthHandler(service.NewAuthService("test-jwt-secret"))
	body, _ := json.Marshal(map[string]string{"email": "buyer@example.com"})
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
