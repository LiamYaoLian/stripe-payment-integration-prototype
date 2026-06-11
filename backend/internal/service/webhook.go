package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/rs/xid"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

var handledEvents = map[stripe.EventType]bool{
	"checkout.session.completed":              true,
	"checkout.session.async_payment_succeeded": true,
	"checkout.session.async_payment_failed":  true,
	"checkout.session.expired":               true,
}

type WebhookService struct {
	store         *db.Store
	webhookSecret string
}

func NewWebhookService(store *db.Store, webhookSecret string) *WebhookService {
	return &WebhookService{store: store, webhookSecret: webhookSecret}
}

type WebhookResult struct {
	StatusCode int
	Body       map[string]any
}

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
		case "processed", "ignored":
			return &WebhookResult{StatusCode: 200, Body: map[string]any{"received": true}}, nil
		case "failed":
			// retry below
		case "received":
			return &WebhookResult{StatusCode: 503, Body: map[string]any{"received": false}}, nil
		}
	} else {
		payload, _ := json.Marshal(event)
		if err := s.store.InsertWebhookEvent(ctx, db.WebhookEvent{
			ID: xid.New().String(), StripeEventID: event.ID, EventType: string(event.Type),
			ProcessingStatus: "received", Payload: payload,
		}); err != nil {
			return nil, err
		}
	}

	if !handledEvents[event.Type] {
		_ = s.store.MarkWebhookIgnored(ctx, event.ID)
		return &WebhookResult{StatusCode: 200, Body: map[string]any{"received": true}}, nil
	}

	claimed, err := s.store.ClaimWebhookEvent(ctx, event.ID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		existing, err = s.store.GetWebhookEvent(ctx, event.ID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return &WebhookResult{StatusCode: 503, Body: map[string]any{"received": false}}, nil
		}
		switch existing.ProcessingStatus {
		case "processed":
			return &WebhookResult{StatusCode: 200, Body: map[string]any{"received": true}}, nil
		case "failed":
			return &WebhookResult{StatusCode: 500, Body: map[string]any{"received": false}}, nil
		default:
			return &WebhookResult{StatusCode: 503, Body: map[string]any{"received": false}}, nil
		}
	}

	if err := s.processEvent(ctx, event); err != nil {
		_ = s.store.FailWebhookEvent(ctx, event.ID)
		slog.Error("webhook processing failed", "event_id", event.ID, "error", err)
		return &WebhookResult{StatusCode: 500, Body: map[string]any{"received": false}}, nil
	}

	return &WebhookResult{StatusCode: 200, Body: map[string]any{"received": true}}, nil
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
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *WebhookService) resolveOrder(ctx context.Context, sessionID string) (*db.Order, error) {
	return s.store.GetOrderBySessionID(ctx, sessionID)
}

func paymentIntentID(sess *stripe.CheckoutSession) *string {
	if sess.PaymentIntent == nil {
		return nil
	}
	if sess.PaymentIntent.ID != "" {
		id := sess.PaymentIntent.ID
		return &id
	}
	return nil
}

func (s *WebhookService) handleSessionCompleted(ctx context.Context, event stripe.Event) error {
	sess, err := parseSession(event)
	if err != nil {
		return err
	}
	if sess.Status != stripe.CheckoutSessionStatusComplete {
		return nil
	}
	order, err := s.resolveOrder(ctx, sess.ID)
	if err != nil || order == nil {
		return err
	}

	pi := paymentIntentID(sess)
	now := time.Now().UTC()

	if sess.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
		_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, "paid", pi, &now, []string{"pending", "processing"})
		return err
	}
	if sess.PaymentStatus == stripe.CheckoutSessionPaymentStatusUnpaid {
		_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, "processing", pi, nil, []string{"pending"})
		return err
	}
	return nil
}

func (s *WebhookService) handleAsyncSucceeded(ctx context.Context, event stripe.Event) error {
	sess, err := parseSession(event)
	if err != nil {
		return err
	}
	order, err := s.resolveOrder(ctx, sess.ID)
	if err != nil || order == nil {
		return err
	}
	pi := paymentIntentID(sess)
	now := time.Now().UTC()
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, "paid", pi, &now, []string{"pending", "processing"})
	return err
}

func (s *WebhookService) handleAsyncFailed(ctx context.Context, event stripe.Event) error {
	sess, err := parseSession(event)
	if err != nil {
		return err
	}
	order, err := s.resolveOrder(ctx, sess.ID)
	if err != nil || order == nil {
		return err
	}
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, "failed", nil, nil, []string{"pending", "processing"})
	return err
}

func (s *WebhookService) handleSessionExpired(ctx context.Context, event stripe.Event) error {
	sess, err := parseSession(event)
	if err != nil {
		return err
	}
	order, err := s.resolveOrder(ctx, sess.ID)
	if err != nil || order == nil {
		return err
	}
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, "expired", nil, nil, []string{"pending"})
	return err
}
