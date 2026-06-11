package api

import (
	"testing"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

func TestNewOrderResponse(t *testing.T) {
	paidAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	response := NewOrderResponse(&db.Order{
		ID: "ord1", OrderNumber: "ORD-1", Status: "paid",
		TotalAmountCents: 4900, Currency: "usd", PaidAt: &paidAt,
		Items: []db.OrderItem{{ProductName: "Pro", Quantity: 1, LineTotalCents: 4900}},
	})
	if response.OrderNumber != "ORD-1" || response.Status != "paid" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.PaidAt == nil || *response.PaidAt != paidAt.Format(time.RFC3339) {
		t.Fatalf("paidAt %+v", response.PaidAt)
	}
}

func TestNewOrderResponseWithoutPaidAt(t *testing.T) {
	response := NewOrderResponse(&db.Order{
		ID: "ord2", OrderNumber: "ORD-2", Status: "pending",
		TotalAmountCents: 1000, Currency: "usd",
	})
	if response.PaidAt != nil {
		t.Fatalf("expected no paidAt, got %v", response.PaidAt)
	}
}

func TestNewProductResponse(t *testing.T) {
	description := "A product"
	response := NewProductResponse(db.Product{
		ID: "p1", Name: "Pro", Description: &description,
		UnitAmountCents: 4900, Currency: "usd",
	})
	if response.ID != "p1" || response.Name != "Pro" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Description == nil || *response.Description != description {
		t.Fatalf("description %+v", response.Description)
	}
}
