package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/middleware"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
)

func TestAuthRegisterAndMe(t *testing.T) {
	store := testutil.NewFakeCustomerStore()
	authSvc := service.NewAuthService(store, "test-jwt-secret")
	h := NewAuthHandler(authSvc)

	body, _ := json.Marshal(map[string]string{
		"email": "buyer@example.com", "password": "password123",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Register(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d body=%s", rec.Code, rec.Body.String())
	}

	var registerEnv struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&registerEnv); err != nil {
		t.Fatal(err)
	}
	if registerEnv.Data.Token == "" || registerEnv.Data.User.Email != "buyer@example.com" {
		t.Fatalf("data %+v", registerEnv.Data)
	}

	meHandler := middleware.RequireUserJWT("test-jwt-secret")(http.HandlerFunc(h.Me))
	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+registerEnv.Data.Token)
	meHandler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status %d body=%s", meRec.Code, meRec.Body.String())
	}
}

func TestAuthLogin(t *testing.T) {
	store := testutil.NewFakeCustomerStore()
	authSvc := service.NewAuthService(store, "test-jwt-secret")
	h := NewAuthHandler(authSvc)

	registerBody, _ := json.Marshal(map[string]string{
		"email": "buyer@example.com", "password": "password123",
	})
	registerRec := httptest.NewRecorder()
	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	h.Register(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register status %d", registerRec.Code)
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email": "buyer@example.com", "password": "password123",
	})
	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	h.Login(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status %d body=%s", loginRec.Code, loginRec.Body.String())
	}

	var loginEnv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginEnv); err != nil {
		t.Fatal(err)
	}
	claims, err := auth.VerifyUserToken("test-jwt-secret", loginEnv.Data.Token)
	if err != nil || claims.Email != "buyer@example.com" {
		t.Fatalf("claims %+v err=%v", claims, err)
	}
}
