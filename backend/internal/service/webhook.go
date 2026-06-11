package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
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
	MarkWebhookProcessed(ctx context.Context, stripeEventID string, orderID *string) error
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

// Handle verifies, deduplicates, and processes a Stripe webhook payload.
// Store failures map to Stripe-facing outcomes; err is only set for programmer errors.
func (s *WebhookService) Handle(ctx context.Context, body []byte, signature string) (WebhookOutcome, error) {
	event, err := webhook.ConstructEventWithOptions(body, signature, s.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		slog.Warn("webhook signature verification failed", "error", err)
		return WebhookOutcomeInvalidSignature, nil
	}

	existing, err := s.store.GetWebhookEvent(ctx, event.ID)
	if err != nil {
		return webhookStoreError(WebhookOutcomeRetryLater, err)
	}

	if existing != nil {
		switch existing.ProcessingStatus {
		case domain.WebhookStatusProcessed, domain.WebhookStatusIgnored:
			return WebhookOutcomeAcknowledged, nil
		case domain.WebhookStatusFailed:
			// retry below
		case domain.WebhookStatusReceived, domain.WebhookStatusProcessing:
			return WebhookOutcomeRetryLater, nil
		}
	} else {
		payload, err := json.Marshal(event)
		if err != nil {
			return webhookStoreError(WebhookOutcomeProcessingFailed, fmt.Errorf("marshal webhook event: %w", err))
		}
		if err := s.store.InsertWebhookEvent(ctx, db.WebhookEvent{
			ID: xid.New().String(), StripeEventID: event.ID, EventType: string(event.Type),
			ProcessingStatus: domain.WebhookStatusReceived, Payload: payload,
		}); err != nil {
			if isUniqueViolation(err) {
				return s.handleExistingAfterInsertRace(ctx, event.ID)
			}
			return webhookStoreError(WebhookOutcomeRetryLater, err)
		}
	}

	if !handledEvents[event.Type] {
		if err := s.store.MarkWebhookIgnored(ctx, event.ID); err != nil {
			slog.Warn("failed to mark webhook ignored", "event_id", event.ID, "error", err)
		}
		return WebhookOutcomeAcknowledged, nil
	}

	claimed, err := s.store.ClaimWebhookEvent(ctx, event.ID)
	if err != nil {
		return webhookStoreError(WebhookOutcomeRetryLater, err)
	}
	if !claimed {
		return s.handleUnclaimedEvent(ctx, event.ID)
	}

	orderID, err := s.processEvent(ctx, event)
	if err != nil {
		if failErr := s.store.FailWebhookEvent(ctx, event.ID); failErr != nil {
			slog.Warn("failed to mark webhook failed", "event_id", event.ID, "error", failErr)
		}
		slog.Error("webhook processing failed", "event_id", event.ID, "error", err)
		return WebhookOutcomeProcessingFailed, nil
	}

	if err := s.store.MarkWebhookProcessed(ctx, event.ID, orderID); err != nil {
		return webhookStoreError(WebhookOutcomeProcessingFailed, err)
	}

	return WebhookOutcomeAcknowledged, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *WebhookService) handleExistingAfterInsertRace(ctx context.Context, eventID string) (WebhookOutcome, error) {
	existing, err := s.store.GetWebhookEvent(ctx, eventID)
	if err != nil {
		return webhookStoreError(WebhookOutcomeRetryLater, err)
	}
	if existing == nil {
		return WebhookOutcomeRetryLater, nil
	}
	switch existing.ProcessingStatus {
	case domain.WebhookStatusProcessed, domain.WebhookStatusIgnored:
		return WebhookOutcomeAcknowledged, nil
	default:
		return WebhookOutcomeRetryLater, nil
	}
}

func (s *WebhookService) handleUnclaimedEvent(ctx context.Context, eventID string) (WebhookOutcome, error) {
	existing, err := s.store.GetWebhookEvent(ctx, eventID)
	if err != nil {
		return webhookStoreError(WebhookOutcomeRetryLater, err)
	}
	if existing == nil {
		return WebhookOutcomeRetryLater, nil
	}
	switch existing.ProcessingStatus {
	case domain.WebhookStatusProcessed:
		return WebhookOutcomeAcknowledged, nil
	case domain.WebhookStatusFailed:
		return WebhookOutcomeProcessingFailed, nil
	case domain.WebhookStatusProcessing, domain.WebhookStatusReceived:
		return WebhookOutcomeRetryLater, nil
	default:
		return WebhookOutcomeRetryLater, nil
	}
}

func (s *WebhookService) processEvent(ctx context.Context, event stripe.Event) (*string, error) {
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
		return nil, nil
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

func (s *WebhookService) handleSessionCompleted(ctx context.Context, event stripe.Event) (*string, error) {
	session, err := parseSession(event)
	if err != nil {
		return nil, err
	}
	if session.Status != stripe.CheckoutSessionStatusComplete {
		return nil, nil
	}
	rawOrder, lookupErr := s.store.GetOrderBySessionID(ctx, session.ID)
	order, err := resolveWebhookOrder(rawOrder, session.ID, lookupErr)
	if err != nil {
		return nil, err
	}

	paymentIntent := paymentIntentID(session)
	now := time.Now().UTC()

	if session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
		_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusPaid, paymentIntent, &now, []string{domain.OrderStatusPending, domain.OrderStatusProcessing})
		return &order.ID, err
	}
	if session.PaymentStatus == stripe.CheckoutSessionPaymentStatusUnpaid {
		_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusProcessing, paymentIntent, nil, []string{domain.OrderStatusPending})
		return &order.ID, err
	}
	return &order.ID, nil
}

func (s *WebhookService) handleAsyncSucceeded(ctx context.Context, event stripe.Event) (*string, error) {
	session, err := parseSession(event)
	if err != nil {
		return nil, err
	}
	rawOrder, lookupErr := s.store.GetOrderBySessionID(ctx, session.ID)
	order, err := resolveWebhookOrder(rawOrder, session.ID, lookupErr)
	if err != nil {
		return nil, err
	}
	paymentIntent := paymentIntentID(session)
	now := time.Now().UTC()
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusPaid, paymentIntent, &now, []string{domain.OrderStatusPending, domain.OrderStatusProcessing})
	return &order.ID, err
}

func (s *WebhookService) handleAsyncFailed(ctx context.Context, event stripe.Event) (*string, error) {
	session, err := parseSession(event)
	if err != nil {
		return nil, err
	}
	rawOrder, lookupErr := s.store.GetOrderBySessionID(ctx, session.ID)
	order, err := resolveWebhookOrder(rawOrder, session.ID, lookupErr)
	if err != nil {
		return nil, err
	}
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusFailed, nil, nil, []string{domain.OrderStatusPending, domain.OrderStatusProcessing})
	return &order.ID, err
}

func (s *WebhookService) handleSessionExpired(ctx context.Context, event stripe.Event) (*string, error) {
	session, err := parseSession(event)
	if err != nil {
		return nil, err
	}
	rawOrder, lookupErr := s.store.GetOrderBySessionID(ctx, session.ID)
	order, err := resolveWebhookOrder(rawOrder, session.ID, lookupErr)
	if err != nil {
		return nil, err
	}
	_, err = s.store.UpdateOrderStatusIfAllowed(ctx, order.ID, domain.OrderStatusExpired, nil, nil, []string{domain.OrderStatusPending})
	return &order.ID, err
}
