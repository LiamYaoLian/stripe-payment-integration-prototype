package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
)

func TestWebhooksHandlerInvalidSignature(t *testing.T) {
	payload, err := os.ReadFile("testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewWebhookService(testutil.NewFakeWebhookStore(), testutil.TestWebhookSecret, true, testutil.TestStripeAPIVersion)
	h := NewWebhooksHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", "t=0,v1=invalid")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWebhooksHandlerCheckoutCompletedPaid(t *testing.T) {
	payload, err := os.ReadFile("testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_completed"] = &db.Order{ID: "ord-http", Status: "pending"}
	svc := service.NewWebhookService(store, testutil.TestWebhookSecret, true, testutil.TestStripeAPIVersion)
	h := NewWebhooksHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", testutil.SignWebhookPayload(t, payload, ""))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data["received"] != true {
		t.Fatalf("data %+v", env.Data)
	}
	if len(store.StatusUpdate) != 1 || store.StatusUpdate[0] != "ord-http:paid" {
		t.Fatalf("updates %v", store.StatusUpdate)
	}
}
