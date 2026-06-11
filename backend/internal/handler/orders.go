package handler

import (
	"errors"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type OrdersHandler struct {
	orders *service.OrderService
}

func NewOrdersHandler(orders *service.OrderService) *OrdersHandler {
	return &OrdersHandler{orders: orders}
}

func (h *OrdersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	order, err := h.orders.GetOrder(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if order == nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}
	api.WriteJSON(w, http.StatusOK, service.OrderToResponse(order))
}

func (h *OrdersHandler) GetBySession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	order, err := h.orders.GetOrderBySession(r.Context(), sessionID)
	if err != nil {
		var appErr *api.AppError
		if errors.As(err, &appErr) {
			api.WriteAppError(w, appErr)
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if order == nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}
	api.WriteJSON(w, http.StatusOK, service.OrderToResponse(order))
}
