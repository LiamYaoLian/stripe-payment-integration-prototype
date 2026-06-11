package handler

import (
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

type webhookResponse struct {
	Received bool `json:"received"`
}

func writeWebhookOutcome(w http.ResponseWriter, outcome service.WebhookOutcome) {
	switch outcome {
	case service.WebhookOutcomeInvalidSignature:
		api.WriteError(w, http.StatusBadRequest, "INVALID_SIGNATURE", "invalid stripe webhook signature")
	case service.WebhookOutcomeAcknowledged:
		api.WriteJSON(w, http.StatusOK, webhookResponse{Received: true})
	case service.WebhookOutcomeRetryLater:
		api.WriteError(w, http.StatusServiceUnavailable, "WEBHOOK_RETRY", "webhook processing deferred")
	case service.WebhookOutcomeProcessingFailed:
		api.WriteError(w, http.StatusInternalServerError, "WEBHOOK_PROCESSING_FAILED", "webhook processing failed")
	default:
		api.WriteError(w, http.StatusInternalServerError, "WEBHOOK_PROCESSING_FAILED", "webhook processing failed")
	}
}
