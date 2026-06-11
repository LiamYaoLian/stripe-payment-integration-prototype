package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
)

func TestCheckoutHandlerInvalidJSON(t *testing.T) {
	h := NewCheckoutHandler(service.NewOrderService(nil, nil, "http://localhost:5173"))
	req := httptest.NewRequest(http.MethodPost, "/api/checkout/sessions", bytes.NewReader([]byte("{")))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCheckoutHandlerSuccess(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	store.Products["p1"] = db.Product{ID: "p1", Name: "Pro", UnitAmountCents: 4900, Currency: "usd", Active: true}
	h := NewCheckoutHandler(service.NewOrderService(store, &testutil.FakeStripe{}, "http://localhost:5173"))
	body, _ := json.Marshal(service.CreateCheckoutInput{
		UIMode: "hosted",
		Items:  []service.CheckoutItemInput{{ProductID: "p1", Quantity: 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/checkout/sessions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCheckoutHandlerValidationError(t *testing.T) {
	h := NewCheckoutHandler(service.NewOrderService(nil, nil, "http://localhost:5173"))
	body, _ := json.Marshal(service.CreateCheckoutInput{UIMode: "bad", Items: []service.CheckoutItemInput{}})
	req := httptest.NewRequest(http.MethodPost, "/api/checkout/sessions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}
