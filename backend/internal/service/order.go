package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
	"github.com/rs/xid"
	"github.com/stripe/stripe-go/v82"
)

// CheckoutItemInput is a single line item in a checkout request.
type CheckoutItemInput struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

// CreateCheckoutInput is the request body for creating a checkout session.
type CreateCheckoutInput struct {
	UIMode        string              `json:"uiMode"`
	Items         []CheckoutItemInput `json:"items"`
	CustomerEmail string              `json:"customerEmail"`
	Metadata      map[string]string   `json:"metadata"`
}

// CheckoutResult is returned after a checkout session is created or replayed.
type CheckoutResult struct {
	OrderID      string `json:"orderId"`
	OrderNumber  string `json:"orderNumber"`
	SessionID    string `json:"sessionId"`
	URL          string `json:"url,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
}

type orderStore interface {
	GetOrderByID(ctx context.Context, id string) (*db.Order, error)
	GetOrderBySessionID(ctx context.Context, sessionID string) (*db.Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, key string) (*db.Order, error)
	ClearOrderIdempotencyKey(ctx context.Context, orderID string) error
	GetProduct(ctx context.Context, id string) (*db.Product, error)
	GetProductsByIDs(ctx context.Context, ids []string) (map[string]db.Product, error)
	CreateOrderWithItems(ctx context.Context, order db.CreateOrderParams, items []db.CreateOrderItemParams) error
	UpdateOrderSession(ctx context.Context, orderID, sessionID, checkoutURL, clientSecret string) error
	CancelOrder(ctx context.Context, orderID, reason string) error
}

type checkoutStripeClient interface {
	CreateCheckoutSession(params *stripe.CheckoutSessionParams, idempotencyKey string) (*stripe.CheckoutSession, error)
	ExpireCheckoutSession(sessionID string) error
}

// OrderService handles order reads and checkout session creation.
type OrderService struct {
	store        orderStore
	stripe       checkoutStripeClient
	frontendURL  string
}

// NewOrderService returns an OrderService backed by the given dependencies.
func NewOrderService(store orderStore, stripe checkoutStripeClient, frontendURL string) *OrderService {
	return &OrderService{
		store:       store,
		stripe:      stripe,
		frontendURL: strings.TrimRight(frontendURL, "/"),
	}
}

// GetOrder returns an order by ID when the access token matches.
func (s *OrderService) GetOrder(ctx context.Context, id string, accessToken string) (*db.Order, error) {
	if strings.TrimSpace(id) == "" {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "invalid order id"}
	}
	order, err := s.store.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}
	if err := verifyOrderAccess(order, accessToken); err != nil {
		return nil, err
	}
	return order, nil
}

// GetOrderBySession returns an order by Stripe checkout session ID when the access token matches.
func (s *OrderService) GetOrderBySession(ctx context.Context, sessionID string, accessToken string) (*db.Order, error) {
	if !strings.HasPrefix(sessionID, "cs_") {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "invalid session id"}
	}
	order, err := s.store.GetOrderBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}
	if err := verifyOrderAccess(order, accessToken); err != nil {
		return nil, err
	}
	return order, nil
}

func verifyOrderAccess(order *db.Order, accessToken string) error {
	hash := ""
	if order.AccessTokenHash != nil {
		hash = *order.AccessTokenHash
	}
	if !auth.VerifyOrderAccessToken(accessToken, hash) {
		return &api.AppError{Status: 401, Code: "UNAUTHORIZED", Message: "invalid or missing order access token"}
	}
	return nil
}

// CreateCheckoutSession validates input, creates a pending order, and starts Stripe checkout.
func (s *OrderService) CreateCheckoutSession(ctx context.Context, idempotencyKey string, input CreateCheckoutInput) (*CheckoutResult, error) {
	if err := validateCheckoutInput(input); err != nil {
		return nil, err
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return nil, err
	}

	bodyHash, err := canonicalBodyHash(input)
	if err != nil {
		return nil, err
	}

	if replay, err := s.resolveIdempotency(ctx, idempotencyKey, bodyHash); err != nil || replay != nil {
		return replay, err
	}

	lineItems, err := s.buildCheckoutLineItems(ctx, input.Items)
	if err != nil {
		return nil, err
	}

	orderID := xid.New().String()
	orderNumber := generateOrderNumber(orderID)
	urls := s.buildCheckoutURLs(input.UIMode)

	accessToken, tokenHash, err := auth.GenerateOrderAccessToken()
	if err != nil {
		return nil, err
	}

	if err := s.insertPendingOrder(ctx, orderID, orderNumber, bodyHash, tokenHash, idempotencyKey, input, lineItems, urls); err != nil {
		return nil, err
	}

	var emailPtr *string
	if input.CustomerEmail != "" {
		emailPtr = &input.CustomerEmail
	}
	params := s.buildStripeSessionParams(orderID, orderNumber, input, lineItems, urls, emailPtr)

	session, err := s.stripe.CreateCheckoutSession(params, orderID)
	if err != nil {
		if cancelErr := s.store.CancelOrder(ctx, orderID, "stripe_api_error"); cancelErr != nil {
			slog.Warn("failed to cancel order after stripe error", "order_id", orderID, "error", cancelErr)
		}
		slog.Error("stripe checkout session failed", "order_id", orderID, "error", err)
		return nil, &api.AppError{Status: 502, Code: "STRIPE_ERROR", Message: "failed to create checkout session"}
	}

	if err := s.persistSessionWithRetry(ctx, orderID, session); err != nil {
		s.compensateFailedCheckout(ctx, orderID, session.ID)
		return nil, &api.AppError{Status: 502, Code: "STRIPE_ERROR", Message: "failed to persist checkout session"}
	}

	return buildCheckoutResult(orderID, orderNumber, input.UIMode, session, accessToken), nil
}

func canonicalBodyHash(input CreateCheckoutInput) (string, error) {
	bytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}

func generateOrderNumber(orderID string) string {
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), strings.ToUpper(orderID))
}

func replayCheckout(order *db.Order) *CheckoutResult {
	result := &CheckoutResult{
		OrderID: order.ID, OrderNumber: order.OrderNumber,
	}
	if order.StripeCheckoutSessionID != nil {
		result.SessionID = *order.StripeCheckoutSessionID
	}
	if order.UIMode == domain.UIModeHosted && order.StripeCheckoutURL != nil {
		result.URL = *order.StripeCheckoutURL
	}
	if order.UIMode == domain.UIModeEmbedded && order.StripeClientSecret != nil {
		result.ClientSecret = *order.StripeClientSecret
	}
	return result
}

func extractRequestHash(meta json.RawMessage) string {
	var metadata map[string]any
	if err := json.Unmarshal(meta, &metadata); err != nil {
		return ""
	}
	value, ok := metadata["_request_body_hash"].(string)
	if !ok {
		return ""
	}
	return value
}
