package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
	"github.com/rs/xid"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

var handledEvents = map[stripe.EventType]bool{
	"checkout.session.completed":               true,
	"checkout.session.async_payment_succeeded": true,
	"checkout.session.async_payment_failed":    true,
	"checkout.session.expired":                 true,
}

type webhookStore interface {
	GetWebhookEvent(ctx context.Context, stripeEventID string) (*db.WebhookEvent, error)
	InsertWebhookEvent(ctx context.Context, e db.WebhookEvent) error
	MarkWebhookIgnored(ctx context.Context, stripeEventID string) error
	ClaimWebhookEvent(ctx context.Context, stripeEventID string) (bool, error)
	FailWebhookEvent(ctx context.Context, stripeEventID string) error
	GetOrderBySessionID(ctx context.Context, sessionID string) (*db.Order, error)
	UpdateOrderStatusIfAllowed(ctx context.Context, orderID, newStatus string, paymentIntentID *string, paidAt *time.Time, allowedFrom []string) (bool, error)
}

// WebhookService processes Stripe webhook events.
type WebhookService struct {
	store         webhookStore
	webhookSecret string
}

// NewWebhookService returns a WebhookService backed by the given store.
func NewWebhookService(store webhookStore, webhookSecret string) *WebhookService {
	return &WebhookService{store: store, webhookSecret: webhookSecret}
}

// WebhookResult carries the HTTP response for a processed webhook.
type WebhookResult struct {
	StatusCode int
	Body       map[string]any
}

// Handle verifies, deduplicates, and processes a Stripe webhook payload.
func (s *WebhookService) Handle(ctx context.Context, body []byte, signature string) (*WebhookResult, error) {
	event, err := webhook.ConstructEventWithOptions(body, signature, s.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		slog.Warn("webhook signature verification failed", "error", err)
		return &WebhookResult{StatusCode: 400, Body: map[string]any{"error": "invalid signature"}}, nil
	}

	existing, err := s.store.GetWebhookEvent(ctx, event.ID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		switch existing.ProcessingStatus {
		case domain.WebhookStatusProcessed, domain.WebhookStatusIgnored:
			return receivedOK(), nil
		case domain.WebhookStatusFailed:
			// retry below
		case domain.WebhookStatusReceived:
			return receivedUnavailable(), nil
		}
	} else {
		payload, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("marshal webhook event: %w", err)
		}
		if err := s.store.InsertWebhookEvent(ctx, db.WebhookEvent{
			ID: xid.New().String(), StripeEventID: event.ID, EventType: string(event.Type),
			ProcessingStatus: domain.WebhookStatusReceived, Payload: payload,
		}); err != nil {
			return nil, err
		}
	}

	if !handledEvents[event.Type] {
		if err := s.store.MarkWebhookIgnored(ctx, event.ID); err != nil {
			slog.Warn("failed to mark webhook ignored", "event_id", event.ID, "error", err)
		}
		return receivedOK(), nil
	}

	claimed, err := s.store.ClaimWebhookEvent(ctx, event.ID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return s.handleUnclaimedEvent(ctx, event.ID)
	}

	if err := s.processEvent(ctx, event); err != nil {
		if failErr := s.store.FailWebhookEvent(ctx, event.ID); failErr != nil {
			slog.Warn("failed to mark webhook failed", "event_id", event.ID, "error", failErr)
		}
		slog.Error("webhook processing failed", "event_id", event.ID, "error", err)
		return receivedFailed(), nil
	}

	return receivedOK(), nil
}

func (s *WebhookService) handleUnclaimedEvent(ctx context.Context, eventID string) (*WebhookResult, error) {
	existing, err := s.store.GetWebhookEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return receivedUnavailable(), nil
	}
	switch existing.ProcessingStatus {
	case domain.WebhookStatusProcessed:
		return receivedOK(), nil
	case domain.WebhookStatusFailed:
		return receivedFailed(), nil
	default:
		return receivedUnavailable(), nil
	}
}

func receivedOK() *WebhookResult {
	return &WebhookResult{StatusCode: 200, Body: map[string]any{"received": true}}
}

func receivedUnavailable() *WebhookResult {
	return &WebhookResult{StatusCode: 503, Body: map[string]any{"received": false}}
}

func receivedFailed() *WebhookResult {
	return &WebhookResult{StatusCode: 500, Body: map[string]any{"received": false}}
}

func (s *WebhookService) processEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "checkout.session.completed":
		return s.handleSessionCompleted(ctx, event)
	case "checkout.session.async_payment_succeeded":
		return s.handleAsyncSucceeded(ctx, event)
	case "checkout.session.async_payment_failed":
		return s.handleAsyncFailed(ctx, event)
	case "checkout.session.expired":
		return s.handleSessionExpired(ctx, event)
	default:
		return nil
	}
}

func parseSession(event stripe.Event) (*stripe.CheckoutSession, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func paymentIntentID(session *stripe.CheckoutSession) *string {
	if session.PaymentIntent == nil || session.PaymentIntent.ID == "" {
		return nil
	}
	id := session.PaymentIntent.ID
	return &id
}

func (s *WebhookService) handleSessionCompleted(ctx context.Context, event stripe.Event) error {
	session, err := parseSession(event)
	if err != nil {
		return err
	}
	if session.Status != stripe.CheckoutSessionStatusComplete {
		return nil
	}
	order, err := s.store.GetOrderBySessionID(ctx, session.ID)
	if err != nil || order == nil {
		return err
	}

	paymentIntent := paymentIntentID(session)
	now := time.Now().UTC()

	if session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
		_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusPaid, paymentIntent, &now, []string{domain.OrderStatusPending, domain.OrderStatusProcessing})
		return err
	}
	if session.PaymentStatus == stripe.CheckoutSessionPaymentStatusUnpaid {
		_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusProcessing, paymentIntent, nil, []string{domain.OrderStatusPending})
		return err
	}
	return nil
}

func (s *WebhookService) handleAsyncSucceeded(ctx context.Context, event stripe.Event) error {
	session, err := parseSession(event)
	if err != nil {
		return err
	}
	order, err := s.store.GetOrderBySessionID(ctx, session.ID)
	if err != nil || order == nil {
		return err
	}
	paymentIntent := paymentIntentID(session)
	now := time.Now().UTC()
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusPaid, paymentIntent, &now, []string{domain.OrderStatusPending, domain.OrderStatusProcessing})
	return err
}

func (s *WebhookService) handleAsyncFailed(ctx context.Context, event stripe.Event) error {
	session, err := parseSession(event)
	if err != nil {
		return err
	}
	order, err := s.store.GetOrderBySessionID(ctx, session.ID)
	if err != nil || order == nil {
		return err
	}
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusFailed, nil, nil, []string{domain.OrderStatusPending, domain.OrderStatusProcessing})
	return err
}

func (s *WebhookService) handleSessionExpired(ctx context.Context, event stripe.Event) error {
	session, err := parseSession(event)
	if err != nil {
		return err
	}
	order, err := s.store.GetOrderBySessionID(ctx, session.ID)
	if err != nil || order == nil {
		return err
	}
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusExpired, nil, nil, []string{domain.OrderStatusPending})
	return err
}
