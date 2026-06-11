package handler

import (
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
)

// ProductsHandler serves the product catalog API.
type ProductsHandler struct {
	products ProductCatalog
}

// NewProductsHandler returns a handler for GET /api/products.
func NewProductsHandler(products ProductCatalog) *ProductsHandler {
	return &ProductsHandler{products: products}
}

func (h *ProductsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	products, err := h.products.ListProducts(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	list := make([]api.ProductResponse, 0, len(products))
	for _, product := range products {
		list = append(list, api.NewProductResponse(product))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"products": list})
}
