package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/metrics"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

// CheckoutHandler serves POST /api/checkout/sessions.
type CheckoutHandler struct {
	orders OrderService
}

// NewCheckoutHandler returns a handler for checkout session creation.
func NewCheckoutHandler(orders OrderService) *CheckoutHandler {
	return &CheckoutHandler{orders: orders}
}

func (h *CheckoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input service.CreateCheckoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	result, err := h.orders.CreateCheckoutSession(r.Context(), idempotencyKey, input)
	recordCheckoutOutcome(err)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, result)
}

func recordCheckoutOutcome(err error) {
	outcome := "success"
	if err != nil {
		var apiErr *api.AppError
		if errors.As(err, &apiErr) {
			outcome = strings.ToLower(apiErr.Code)
		} else {
			outcome = "error"
		}
	}
	metrics.CheckoutSessionsTotal.WithLabelValues(outcome).Inc()
}
