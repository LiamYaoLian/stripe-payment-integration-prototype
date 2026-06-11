package handler

import (
	"io"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
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
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "failed to read body")
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	outcome, err := h.webhooks.Handle(r.Context(), body, signature)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeWebhookOutcome(w, outcome)
}
