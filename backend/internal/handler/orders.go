package handler

import (
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

// OrdersHandler serves order lookup endpoints.
type OrdersHandler struct {
	orders *service.OrderService
}

// NewOrdersHandler returns a handler for order routes.
func NewOrdersHandler(orders *service.OrderService) *OrdersHandler {
	return &OrdersHandler{orders: orders}
}

func orderAccessToken(r *http.Request) string {
	return r.Header.Get("X-Order-Token")
}

func (h *OrdersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	order, err := h.orders.GetOrder(r.Context(), id, orderAccessToken(r))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if order == nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}
	api.WriteJSON(w, http.StatusOK, api.NewOrderResponse(order))
}

func (h *OrdersHandler) GetBySession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	order, err := h.orders.GetOrderBySession(r.Context(), sessionID, orderAccessToken(r))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if order == nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "order not found")
		return
	}
	api.WriteJSON(w, http.StatusOK, api.NewOrderResponse(order))
}
