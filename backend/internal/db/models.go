package db

import (
	"encoding/json"
	"time"
)

// Product is a sellable catalog item.
type Product struct {
	ID              string
	Name            string
	Description     *string
	UnitAmountCents int32
	Currency        string
	Active          bool
}

// OrderItem is a line item on an order.
type OrderItem struct {
	ID              string
	OrderID         string
	ProductID       *string
	ProductName     string
	Quantity        int32
	UnitAmountCents int32
	LineTotalCents  int32
}

// Order is a customer purchase record.
type Order struct {
	ID                      string
	OrderNumber             string
	IdempotencyKey          *string
	Status                  string
	TotalAmountCents        int32
	Currency                string
	CustomerEmail           *string
	StripeCheckoutSessionID *string
	StripePaymentIntentID   *string
	StripeCheckoutURL       *string
	StripeClientSecret      *string
	UIMode                  string
	SuccessURL              *string
	CancelURL               *string
	ReturnURL               *string
	Metadata                json.RawMessage
	PaidAt                  *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
	AccessTokenHash         *string
	Items                   []OrderItem
}

// WebhookEvent records a received Stripe webhook for idempotent processing.
type WebhookEvent struct {
	ID               string
	StripeEventID    string
	EventType        string
	OrderID          *string
	ProcessingStatus string
	Payload          json.RawMessage
	ProcessedAt          *time.Time
	ProcessingStartedAt  *time.Time
}

// CreateOrderParams holds fields for inserting a new order.
type CreateOrderParams struct {
	ID               string
	OrderNumber      string
	IdempotencyKey   *string
	TotalAmountCents int32
	Currency         string
	CustomerEmail    *string
	UIMode           string
	SuccessURL       *string
	CancelURL        *string
	ReturnURL        *string
	Metadata         json.RawMessage
	RequestBodyHash  string
	AccessTokenHash  string
}

// CreateOrderItemParams holds fields for inserting an order line item.
type CreateOrderItemParams struct {
	ID              string
	OrderID         string
	ProductID       string
	ProductName     string
	Quantity        int32
	UnitAmountCents int32
	LineTotalCents  int32
}
