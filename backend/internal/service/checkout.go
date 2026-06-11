package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
	"github.com/rs/xid"
	"github.com/stripe/stripe-go/v82"
)

type checkoutLineItems struct {
	currency string
	total    int32
	stripe   []*stripe.CheckoutSessionLineItemParams
	db       []db.CreateOrderItemParams
}

type checkoutURLs struct {
	successURL *string
	cancelURL  *string
	returnURL  *string
}

func (s *OrderService) resolveIdempotency(
	ctx context.Context,
	idempotencyKey string,
	bodyHash string,
) (*CheckoutResult, error) {
	if idempotencyKey == "" {
		return nil, nil
	}
	existing, err := s.store.GetOrderByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	existingHash := extractRequestHash(existing.Metadata)
	if existingHash != "" && existingHash != bodyHash {
		return nil, &api.AppError{Status: 409, Code: "IDEMPOTENCY_CONFLICT", Message: "idempotency key reused with different body"}
	}
	if existing.Status == domain.OrderStatusCanceled {
		if err := s.store.ClearOrderIdempotencyKey(ctx, existing.ID); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if existing.StripeCheckoutSessionID == nil || *existing.StripeCheckoutSessionID == "" {
		return nil, &api.AppError{Status: 409, Code: "CHECKOUT_IN_PROGRESS", Message: "checkout session creation in progress"}
	}
	accessToken, tokenHash, err := auth.GenerateOrderAccessToken()
	if err != nil {
		return nil, err
	}
	if err := s.store.UpdateOrderAccessTokenHash(ctx, existing.ID, tokenHash); err != nil {
		return nil, err
	}
	return replayCheckout(existing, accessToken), nil
}

func (s *OrderService) buildCheckoutLineItems(ctx context.Context, items []CheckoutItemInput) (*checkoutLineItems, error) {
	result := &checkoutLineItems{
		stripe: make([]*stripe.CheckoutSessionLineItemParams, 0, len(items)),
		db:     make([]db.CreateOrderItemParams, 0, len(items)),
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProductID)
	}
	products, err := s.store.GetProductsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for index, item := range items {
		product, ok := products[item.ProductID]
		if !ok {
			return nil, &api.AppError{Status: 404, Code: "PRODUCT_NOT_FOUND", Message: fmt.Sprintf("product not found: %s", item.ProductID)}
		}
		if index == 0 {
			result.currency = product.Currency
		} else if product.Currency != result.currency {
			return nil, &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "mixed currency not allowed"}
		}

		lineTotal := product.UnitAmountCents * item.Quantity
		result.total += lineTotal

		productData := &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
			Name: stripe.String(product.Name),
		}
		if product.Description != nil && *product.Description != "" {
			productData.Description = stripe.String(*product.Description)
		}

		result.stripe = append(result.stripe, &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:    stripe.String(product.Currency),
				UnitAmount:  stripe.Int64(int64(product.UnitAmountCents)),
				ProductData: productData,
			},
			Quantity: stripe.Int64(int64(item.Quantity)),
		})

		result.db = append(result.db, db.CreateOrderItemParams{
			ID:              xid.New().String(),
			ProductID:       product.ID,
			ProductName:     product.Name,
			Quantity:        item.Quantity,
			UnitAmountCents: product.UnitAmountCents,
			LineTotalCents:  lineTotal,
		})
	}
	return result, nil
}

func (s *OrderService) buildCheckoutURLs(uiMode string) checkoutURLs {
	if uiMode == domain.UIModeHosted {
		success := fmt.Sprintf("%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", s.frontendURL)
		cancel := fmt.Sprintf("%s/checkout/cancel", s.frontendURL)
		return checkoutURLs{successURL: &success, cancelURL: &cancel}
	}
	returnURL := fmt.Sprintf("%s/checkout/complete", s.frontendURL)
	return checkoutURLs{returnURL: &returnURL}
}

func marshalMetadata(metadata map[string]string) (json.RawMessage, error) {
	meta := make(map[string]string, len(metadata))
	for key, value := range metadata {
		meta[key] = value
	}
	bytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return bytes, nil
}

func (s *OrderService) insertPendingOrder(
	ctx context.Context,
	orderID, orderNumber, bodyHash, accessTokenHash string,
	idempotencyKey string,
	input CreateCheckoutInput,
	lineItems *checkoutLineItems,
	urls checkoutURLs,
) error {
	metaBytes, err := marshalMetadata(input.Metadata)
	if err != nil {
		return err
	}

	var idemPtr *string
	if idempotencyKey != "" {
		idemPtr = &idempotencyKey
	}
	var emailPtr *string
	if input.CustomerEmail != "" {
		emailPtr = &input.CustomerEmail
	}

	dbItems := lineItems.db
	for index := range dbItems {
		dbItems[index].OrderID = orderID
	}

	return s.store.CreateOrderWithItems(ctx, db.CreateOrderParams{
		ID: orderID, OrderNumber: orderNumber, IdempotencyKey: idemPtr,
		TotalAmountCents: lineItems.total, Currency: lineItems.currency, CustomerEmail: emailPtr,
		UIMode: input.UIMode, SuccessURL: urls.successURL, CancelURL: urls.cancelURL, ReturnURL: urls.returnURL,
		Metadata: metaBytes, RequestBodyHash: bodyHash, AccessTokenHash: accessTokenHash,
	}, dbItems)
}

func (s *OrderService) buildStripeSessionParams(
	orderID, orderNumber string,
	input CreateCheckoutInput,
	lineItems *checkoutLineItems,
	urls checkoutURLs,
	emailPtr *string,
) *stripe.CheckoutSessionParams {
	stripeMeta := map[string]string{"order_id": orderID, "order_number": orderNumber}
	for key, value := range input.Metadata {
		stripeMeta[key] = value
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		UIMode:            stripe.String(domain.StripeUIMode(input.UIMode)),
		LineItems:         lineItems.stripe,
		ClientReferenceID: stripe.String(orderNumber),
		Metadata:          stripeMeta,
	}
	if emailPtr != nil {
		params.CustomerEmail = stripe.String(*emailPtr)
	}
	if input.UIMode == domain.UIModeHosted {
		params.SuccessURL = stripe.String(*urls.successURL)
		params.CancelURL = stripe.String(*urls.cancelURL)
	} else {
		params.ReturnURL = stripe.String(*urls.returnURL)
	}
	return params
}

func (s *OrderService) persistSessionWithRetry(
	ctx context.Context,
	orderID string,
	session *stripe.CheckoutSession,
) error {
	checkoutURL := session.URL
	clientSecret := session.ClientSecret

	var persistErr error
	for attempt := 0; attempt < 3; attempt++ {
		persistErr = s.store.UpdateOrderSession(ctx, orderID, session.ID, checkoutURL, clientSecret)
		if persistErr == nil {
			return nil
		}
		if err := sleepWithContext(ctx, time.Duration(attempt+1)*50*time.Millisecond); err != nil {
			return err
		}
	}
	return persistErr
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *OrderService) compensateFailedCheckout(ctx context.Context, orderID, sessionID string) {
	if err := s.stripe.ExpireCheckoutSession(sessionID); err != nil {
		slog.Warn("failed to expire stripe session", "session_id", sessionID, "error", err)
	}
	if err := s.store.CancelOrder(ctx, orderID, "persist_failed"); err != nil {
		slog.Warn("failed to cancel order after persist failure", "order_id", orderID, "error", err)
	}
}

func buildCheckoutResult(orderID, orderNumber string, uiMode string, session *stripe.CheckoutSession, accessToken string) *CheckoutResult {
	result := &CheckoutResult{
		OrderID: orderID, OrderNumber: orderNumber, SessionID: session.ID, AccessToken: accessToken,
	}
	if uiMode == domain.UIModeHosted {
		result.URL = session.URL
	} else {
		result.ClientSecret = session.ClientSecret
	}
	return result
}
