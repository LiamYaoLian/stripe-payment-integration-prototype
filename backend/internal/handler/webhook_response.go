package handler

import (
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

func writeWebhookOutcome(w http.ResponseWriter, outcome service.WebhookOutcome) {
	switch outcome {
	case service.WebhookOutcomeInvalidSignature:
		api.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
	case service.WebhookOutcomeAcknowledged:
		api.WriteJSON(w, http.StatusOK, map[string]bool{"received": true})
	case service.WebhookOutcomeRetryLater:
		api.WriteJSON(w, http.StatusServiceUnavailable, map[string]bool{"received": false})
	case service.WebhookOutcomeProcessingFailed:
		api.WriteJSON(w, http.StatusInternalServerError, map[string]bool{"received": false})
	default:
		api.WriteJSON(w, http.StatusInternalServerError, map[string]bool{"received": false})
	}
}
