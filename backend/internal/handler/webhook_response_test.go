package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

func TestWriteWebhookOutcomeMapsToHTTP(t *testing.T) {
	tests := []struct {
		name       string
		outcome    service.WebhookOutcome
		wantStatus int
		wantKey    string
		wantValue  any
	}{
		{name: "invalid signature", outcome: service.WebhookOutcomeInvalidSignature, wantStatus: 400, wantKey: "error", wantValue: "invalid signature"},
		{name: "acknowledged", outcome: service.WebhookOutcomeAcknowledged, wantStatus: 200, wantKey: "received", wantValue: true},
		{name: "retry later", outcome: service.WebhookOutcomeRetryLater, wantStatus: 503, wantKey: "received", wantValue: false},
		{name: "processing failed", outcome: service.WebhookOutcomeProcessingFailed, wantStatus: 500, wantKey: "received", wantValue: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeWebhookOutcome(rec, tc.outcome)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
			}
			var env struct {
				Data map[string]any `json:"data"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
				t.Fatal(err)
			}
			if env.Data[tc.wantKey] != tc.wantValue {
				t.Fatalf("data %+v", env.Data)
			}
		})
	}
}
