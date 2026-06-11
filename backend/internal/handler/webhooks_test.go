package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

func TestWebhooksHandlerInvalidSignature(t *testing.T) {
	payload, err := os.ReadFile("testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewWebhookService(nil, "whsec_test_secret")
	h := NewWebhooksHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", "t=0,v1=invalid")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
