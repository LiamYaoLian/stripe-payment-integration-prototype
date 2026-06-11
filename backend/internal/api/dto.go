package api

import (
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

// ProductResponse is the JSON representation of a catalog product.
type ProductResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	UnitAmountCents int32   `json:"unitAmountCents"`
	Currency        string  `json:"currency"`
}

// OrderItemResponse is a line item on an order.
type OrderItemResponse struct {
	ProductName    string `json:"productName"`
	Quantity       int32  `json:"quantity"`
	LineTotalCents int32  `json:"lineTotalCents"`
}

// OrderResponse is the JSON representation of an order.
type OrderResponse struct {
	ID               string              `json:"id"`
	OrderNumber      string              `json:"orderNumber"`
	Status           string              `json:"status"`
	TotalAmountCents int32               `json:"totalAmountCents"`
	Currency         string              `json:"currency"`
	PaidAt           *string             `json:"paidAt,omitempty"`
	Items            []OrderItemResponse `json:"items"`
}

// NewProductResponse maps a database product to its API representation.
func NewProductResponse(product db.Product) ProductResponse {
	return ProductResponse{
		ID:              product.ID,
		Name:            product.Name,
		Description:     product.Description,
		UnitAmountCents: product.UnitAmountCents,
		Currency:        product.Currency,
	}
}

// NewOrderResponse maps a database order to its API representation.
func NewOrderResponse(order *db.Order) OrderResponse {
	items := make([]OrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, OrderItemResponse{
			ProductName:    item.ProductName,
			Quantity:       item.Quantity,
			LineTotalCents: item.LineTotalCents,
		})
	}
	response := OrderResponse{
		ID:               order.ID,
		OrderNumber:      order.OrderNumber,
		Status:           order.Status,
		TotalAmountCents: order.TotalAmountCents,
		Currency:         order.Currency,
		Items:            items,
	}
	if order.PaidAt != nil {
		formatted := order.PaidAt.UTC().Format(time.RFC3339)
		response.PaidAt = &formatted
	}
	return response
}
