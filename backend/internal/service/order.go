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
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/stripeclient"
	"github.com/rs/xid"
	"github.com/stripe/stripe-go/v82"
)

type CheckoutItemInput struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

type CreateCheckoutInput struct {
	UIMode        string            `json:"uiMode"`
	Items         []CheckoutItemInput `json:"items"`
	CustomerEmail string            `json:"customerEmail"`
	Metadata      map[string]string `json:"metadata"`
}

type CheckoutResult struct {
	OrderID      string `json:"orderId"`
	OrderNumber  string `json:"orderNumber"`
	SessionID    string `json:"sessionId"`
	URL          string `json:"url,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

type OrderService struct {
	store  *db.Store
	stripe *stripeclient.Client
	front  string
}

func NewOrderService(store *db.Store, stripe *stripeclient.Client, frontendURL string) *OrderService {
	return &OrderService{store: store, stripe: stripe, front: strings.TrimRight(frontendURL, "/")}
}

func (s *OrderService) ListProducts(ctx context.Context) ([]db.Product, error) {
	return s.store.ListActiveProducts(ctx)
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (*db.Order, error) {
	return s.store.GetOrderByID(ctx, id)
}

func (s *OrderService) GetOrderBySession(ctx context.Context, sessionID string) (*db.Order, error) {
	if !strings.HasPrefix(sessionID, "cs_") {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "invalid session id"}
	}
	return s.store.GetOrderBySessionID(ctx, sessionID)
}

func canonicalBodyHash(input CreateCheckoutInput) (string, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func (s *OrderService) CreateCheckoutSession(ctx context.Context, idempotencyKey string, input CreateCheckoutInput) (*CheckoutResult, error) {
	if input.UIMode != "hosted" && input.UIMode != "embedded" {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "uiMode must be hosted or embedded"}
	}
	if len(input.Items) == 0 {
		return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "items required"}
	}
	if err := validateMetadata(input.Metadata); err != nil {
		return nil, err
	}

	bodyHash, err := canonicalBodyHash(input)
	if err != nil {
		return nil, err
	}

	if idempotencyKey != "" {
		existing, err := s.store.GetOrderByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			existingHash := extractRequestHash(existing.Metadata)
			if existingHash != "" && existingHash != bodyHash {
				return nil, &api.AppError{Status: 409, Code: "IDEMPOTENCY_CONFLICT", Message: "idempotency key reused with different body"}
			}
			if existing.Status == "canceled" {
				// allow new order below
			} else if existing.StripeCheckoutSessionID == nil || *existing.StripeCheckoutSessionID == "" {
				return nil, &api.AppError{Status: 409, Code: "CHECKOUT_IN_PROGRESS", Message: "checkout session creation in progress"}
			} else {
				return replayCheckout(existing), nil
			}
		}
	}

	var currency string
	var total int32
	var lineItems []*stripe.CheckoutSessionLineItemParams
	var dbItems []db.CreateOrderItemParams

	for i, item := range input.Items {
		if item.Quantity < 1 {
			return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "quantity must be positive"}
		}
		product, err := s.store.GetProduct(ctx, item.ProductID)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, &api.AppError{Status: 404, Code: "PRODUCT_NOT_FOUND", Message: fmt.Sprintf("product not found: %s", item.ProductID)}
		}
		if i == 0 {
			currency = product.Currency
		} else if product.Currency != currency {
			return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "mixed currency not allowed"}
		}
		lineTotal := product.UnitAmountCents * item.Quantity
		total += lineTotal

		desc := ""
		if product.Description != nil {
			desc = *product.Description
		}
		_ = desc

		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(product.Currency),
				UnitAmount: stripe.Int64(int64(product.UnitAmountCents)),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(product.Name),
				},
			},
			Quantity: stripe.Int64(int64(item.Quantity)),
		})

		dbItems = append(dbItems, db.CreateOrderItemParams{
			ID:              xid.New().String(),
			ProductID:       product.ID,
			ProductName:     product.Name,
			Quantity:        item.Quantity,
			UnitAmountCents: product.UnitAmountCents,
			LineTotalCents:  lineTotal,
		})
	}

	orderID := xid.New().String()
	orderNumber := fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), strings.ToUpper(xid.New().String()[:4]))

	var successURL, cancelURL, returnURL *string
	if input.UIMode == "hosted" {
		su := fmt.Sprintf("%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", s.front)
		cu := fmt.Sprintf("%s/checkout/cancel", s.front)
		successURL = &su
		cancelURL = &cu
	} else {
		ru := fmt.Sprintf("%s/checkout/complete", s.front)
		returnURL = &ru
	}

	meta := map[string]string{}
	for k, v := range input.Metadata {
		meta[k] = v
	}
	metaBytes, _ := json.Marshal(meta)

	var idemPtr *string
	if idempotencyKey != "" {
		idemPtr = &idempotencyKey
	}
	var emailPtr *string
	if input.CustomerEmail != "" {
		emailPtr = &input.CustomerEmail
	}

	for i := range dbItems {
		dbItems[i].OrderID = orderID
	}

	if err := s.store.CreateOrderWithItems(ctx, db.CreateOrderParams{
		ID: orderID, OrderNumber: orderNumber, IdempotencyKey: idemPtr,
		TotalAmountCents: total, Currency: currency, CustomerEmail: emailPtr,
		UIMode: input.UIMode, SuccessURL: successURL, CancelURL: cancelURL, ReturnURL: returnURL,
		Metadata: metaBytes, RequestBodyHash: bodyHash,
	}, dbItems); err != nil {
		return nil, err
	}

	stripeMeta := map[string]string{"order_id": orderID, "order_number": orderNumber}
	for k, v := range input.Metadata {
		stripeMeta[k] = v
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		UIMode:            stripe.String(input.UIMode),
		LineItems:         lineItems,
		ClientReferenceID: stripe.String(orderNumber),
		Metadata:          stripeMeta,
	}
	if emailPtr != nil {
		params.CustomerEmail = stripe.String(*emailPtr)
	}
	if input.UIMode == "hosted" {
		params.SuccessURL = stripe.String(*successURL)
		params.CancelURL = stripe.String(*cancelURL)
	} else {
		params.ReturnURL = stripe.String(*returnURL)
	}

	sess, err := s.stripe.CreateCheckoutSession(params)
	if err != nil {
		_ = s.store.CancelOrder(ctx, orderID, "stripe_api_error")
		slog.Error("stripe checkout session failed", "order_id", orderID, "error", err)
		return nil, &api.AppError{Status: 502, Code: "STRIPE_ERROR", Message: "failed to create checkout session"}
	}

	checkoutURL := ""
	clientSecret := ""
	if sess.URL != "" {
		checkoutURL = sess.URL
	}
	if sess.ClientSecret != "" {
		clientSecret = sess.ClientSecret
	}

	var persistErr error
	for attempt := 0; attempt < 3; attempt++ {
		persistErr = s.store.UpdateOrderSession(ctx, orderID, sess.ID, checkoutURL, clientSecret)
		if persistErr == nil {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	if persistErr != nil {
		_ = s.stripe.ExpireCheckoutSession(sess.ID)
		_ = s.store.CancelOrder(ctx, orderID, "persist_failed")
		return nil, &api.AppError{Status: 502, Code: "STRIPE_ERROR", Message: "failed to persist checkout session"}
	}

	result := &CheckoutResult{
		OrderID: orderID, OrderNumber: orderNumber, SessionID: sess.ID,
	}
	if input.UIMode == "hosted" {
		result.URL = checkoutURL
	} else {
		result.ClientSecret = clientSecret
	}
	return result, nil
}

func replayCheckout(o *db.Order) *CheckoutResult {
	r := &CheckoutResult{
		OrderID: o.ID, OrderNumber: o.OrderNumber,
	}
	if o.StripeCheckoutSessionID != nil {
		r.SessionID = *o.StripeCheckoutSessionID
	}
	if o.UIMode == "hosted" && o.StripeCheckoutURL != nil {
		r.URL = *o.StripeCheckoutURL
	}
	if o.UIMode == "embedded" && o.StripeClientSecret != nil {
		r.ClientSecret = *o.StripeClientSecret
	}
	return r
}

func extractRequestHash(meta json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(meta, &m); err != nil {
		return ""
	}
	if v, ok := m["_request_body_hash"].(string); ok {
		return v
	}
	return ""
}

func validateMetadata(meta map[string]string) error {
	if len(meta) > 50 {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "metadata exceeds 50 keys"}
	}
	for _, v := range meta {
		if len(v) > 500 {
			return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "metadata value exceeds 500 chars"}
		}
	}
	return nil
}

func OrderToResponse(o *db.Order) map[string]any {
	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, map[string]any{
			"productName":    it.ProductName,
			"quantity":       it.Quantity,
			"lineTotalCents": it.LineTotalCents,
		})
	}
	resp := map[string]any{
		"id":               o.ID,
		"orderNumber":      o.OrderNumber,
		"status":           o.Status,
		"totalAmountCents": o.TotalAmountCents,
		"currency":         o.Currency,
		"items":            items,
	}
	if o.PaidAt != nil {
		resp["paidAt"] = o.PaidAt.UTC().Format(time.RFC3339)
	}
	return resp
}

func ProductToResponse(p db.Product) map[string]any {
	resp := map[string]any{
		"id":              p.ID,
		"name":            p.Name,
		"unitAmountCents": p.UnitAmountCents,
		"currency":        p.Currency,
	}
	if p.Description != nil {
		resp["description"] = *p.Description
	}
	return resp
}
