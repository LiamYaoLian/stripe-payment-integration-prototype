package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

type CheckoutHandler struct {
	orders *service.OrderService
}

func NewCheckoutHandler(orders *service.OrderService) *CheckoutHandler {
	return &CheckoutHandler{orders: orders}
}

func (h *CheckoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input service.CreateCheckoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	result, err := h.orders.CreateCheckoutSession(r.Context(), idempotencyKey, input)
	if err != nil {
		var appErr *api.AppError
		if errors.As(err, &appErr) {
			api.WriteAppError(w, appErr)
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	api.WriteJSON(w, http.StatusCreated, result)
}
