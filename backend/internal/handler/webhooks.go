package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/metrics"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

// WebhooksHandler serves POST /api/webhooks/stripe.
type WebhooksHandler struct {
	webhooks *service.WebhookService
}

// NewWebhooksHandler returns a handler for Stripe webhooks.
func NewWebhooksHandler(webhooks *service.WebhookService) *WebhooksHandler {
	return &WebhooksHandler{webhooks: webhooks}
}

func (h *WebhooksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "failed to read body")
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	eventType := parseWebhookEventType(body)
	outcome, err := h.webhooks.Handle(r.Context(), body, signature)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	recordWebhookMetric(eventType, outcome)
	writeWebhookOutcome(w, outcome)
}

func parseWebhookEventType(body []byte) string {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Type == "" {
		return "unknown"
	}
	return envelope.Type
}

func recordWebhookMetric(eventType string, outcome service.WebhookOutcome) {
	metrics.WebhookEventsTotal.WithLabelValues(eventType, webhookOutcomeLabel(outcome)).Inc()
}

func webhookOutcomeLabel(outcome service.WebhookOutcome) string {
	switch outcome {
	case service.WebhookOutcomeAcknowledged:
		return "acknowledged"
	case service.WebhookOutcomeInvalidSignature:
		return "invalid_signature"
	case service.WebhookOutcomeRetryLater:
		return "retry_later"
	case service.WebhookOutcomeProcessingFailed:
		return "processing_failed"
	default:
		return "unknown"
	}
}
