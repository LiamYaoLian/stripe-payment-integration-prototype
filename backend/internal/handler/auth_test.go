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
	store := testutil.NewFakeAuthStore()
	authSvc := service.NewAuthService(store, store, store, "http://localhost:5173", "development")
	h := NewAuthHandler(authSvc, false)

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

	cookie := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookie {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" || !sessionCookie.HttpOnly {
		t.Fatalf("cookies %+v", cookie)
	}

	meHandler := middleware.RequireUserSession(store)(http.HandlerFunc(h.Me))
	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.AddCookie(sessionCookie)
	meHandler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status %d body=%s", meRec.Code, meRec.Body.String())
	}
}

func TestAuthLogin(t *testing.T) {
	store := testutil.NewFakeAuthStore()
	authSvc := service.NewAuthService(store, store, store, "http://localhost:5173", "development")
	h := NewAuthHandler(authSvc, false)

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

	var hasSession bool
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == auth.SessionCookieName && c.Value != "" {
			hasSession = true
		}
	}
	if !hasSession {
		t.Fatal("expected session cookie")
	}
}
