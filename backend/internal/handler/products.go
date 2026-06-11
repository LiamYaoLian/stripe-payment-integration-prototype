package handler

import (
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

type ProductsHandler struct {
	orders *service.OrderService
}

func NewProductsHandler(orders *service.OrderService) *ProductsHandler {
	return &ProductsHandler{orders: orders}
}

func (h *ProductsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	products, err := h.orders.ListProducts(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	list := make([]map[string]any, 0, len(products))
	for _, p := range products {
		list = append(list, service.ProductToResponse(p))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"products": list})
}
