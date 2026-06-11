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
		wantCode   string
		wantData   map[string]any
	}{
		{
			name: "invalid signature", outcome: service.WebhookOutcomeInvalidSignature,
			wantStatus: 400, wantCode: "INVALID_SIGNATURE",
		},
		{
			name: "acknowledged", outcome: service.WebhookOutcomeAcknowledged,
			wantStatus: 200, wantData: map[string]any{"received": true},
		},
		{
			name: "retry later", outcome: service.WebhookOutcomeRetryLater,
			wantStatus: 503, wantCode: "WEBHOOK_RETRY",
		},
		{
			name: "processing failed", outcome: service.WebhookOutcomeProcessingFailed,
			wantStatus: 500, wantCode: "WEBHOOK_PROCESSING_FAILED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeWebhookOutcome(rec, tc.outcome)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
			}
			var env struct {
				Data  map[string]any `json:"data"`
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
				t.Fatal(err)
			}
			if tc.wantCode != "" {
				if env.Error == nil || env.Error.Code != tc.wantCode {
					t.Fatalf("error %+v want code %s", env.Error, tc.wantCode)
				}
				return
			}
			if env.Error != nil {
				t.Fatalf("unexpected error %+v", env.Error)
			}
			for key, want := range tc.wantData {
				if env.Data[key] != want {
					t.Fatalf("data %+v", env.Data)
				}
			}
		})
	}
}
