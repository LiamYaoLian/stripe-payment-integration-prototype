package handler

import (
	"context"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

// ProductCatalog lists active sellable products.
type ProductCatalog interface {
	ListProducts(ctx context.Context) ([]db.Product, error)
}

// OrderService covers checkout creation and authenticated order reads.
type OrderService interface {
	CreateCheckoutSession(ctx context.Context, idempotencyKey string, input service.CreateCheckoutInput) (*service.CheckoutResult, error)
	GetOrder(ctx context.Context, id string, accessToken string) (*db.Order, error)
	GetOrderBySession(ctx context.Context, sessionID string, accessToken string) (*db.Order, error)
	ListOrdersForUser(ctx context.Context, customerID string) ([]db.Order, error)
}

// WebhookProcessor handles Stripe webhook payloads.
type WebhookProcessor interface {
	Handle(ctx context.Context, body []byte, signature string) (service.WebhookOutcome, error)
}

// UserAuthenticator handles user registration and login.
type UserAuthenticator interface {
	Register(ctx context.Context, email, password string) (*service.UserSessionResult, error)
	Login(ctx context.Context, email, password string) (*service.UserSessionResult, error)
	GetUser(ctx context.Context, customerID string) (*service.UserProfile, error)
}
