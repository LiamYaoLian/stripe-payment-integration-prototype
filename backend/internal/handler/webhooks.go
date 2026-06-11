package handler

import (
	"io"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

type WebhooksHandler struct {
	webhooks *service.WebhookService
}

func NewWebhooksHandler(webhooks *service.WebhookService) *WebhooksHandler {
	return &WebhooksHandler{webhooks: webhooks}
}

func (h *WebhooksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "failed to read body")
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	result, err := h.webhooks.Handle(r.Context(), body, sig)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	api.WriteJSON(w, result.StatusCode, result.Body)
}
