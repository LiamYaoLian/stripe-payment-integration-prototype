package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

func TestOrdersGetBySessionInvalidID(t *testing.T) {
	h := NewOrdersHandler(service.NewOrderService(nil, nil, "http://localhost:5173"))
	r := chi.NewRouter()
	r.Get("/api/orders/by-session/{sessionId}", h.GetBySession)

	req := httptest.NewRequest(http.MethodGet, "/api/orders/by-session/not-a-session", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var env api.Envelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}
