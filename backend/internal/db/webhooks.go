package db

import (
	"context"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

const staleWebhookProcessing = 5 * time.Minute

// WebhookCompletion holds the atomic result of processing a webhook event.
type WebhookCompletion struct {
	StripeEventID   string
	OrderID         *string
	NewStatus       string
	PaymentIntentID *string
	PaidAt          *time.Time
	AllowedFrom     []string
}

// GetWebhookEvent returns a webhook event by Stripe event ID.
func (s *Store) GetWebhookEvent(ctx context.Context, stripeEventID string) (*WebhookEvent, error) {
	var event WebhookEvent
	err := s.pool.QueryRow(ctx, `
		SELECT id, stripe_event_id, event_type, order_id, processing_status::text, payload, processed_at, processing_started_at
		FROM webhook_events WHERE stripe_event_id = $1`, stripeEventID).
		Scan(&event.ID, &event.StripeEventID, &event.EventType, &event.OrderID, &event.ProcessingStatus, &event.Payload, &event.ProcessedAt, &event.ProcessingStartedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// InsertWebhookEvent records a newly received webhook event.
func (s *Store) InsertWebhookEvent(ctx context.Context, event WebhookEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_events (id, stripe_event_id, event_type, order_id, processing_status, payload)
		VALUES ($1, $2, $3, $4, $5::webhook_processing_status, $6)`,
		event.ID, event.StripeEventID, event.EventType, event.OrderID, event.ProcessingStatus, event.Payload)
	return err
}

// ClaimWebhookEvent moves a webhook event to processing, reclaiming stale processing rows.
func (s *Store) ClaimWebhookEvent(ctx context.Context, stripeEventID string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = $2::webhook_processing_status,
			processing_started_at = now()
		WHERE stripe_event_id = $1 AND (
			processing_status::text = ANY($3)
			OR (
				processing_status::text = $2
				AND COALESCE(processing_started_at, created_at) < now() - $4::interval
			)
		)`,
		stripeEventID, domain.WebhookStatusProcessing,
		[]string{domain.WebhookStatusReceived, domain.WebhookStatusFailed},
		staleWebhookProcessing.String(),
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CompleteWebhookProcessing atomically updates order state and marks the webhook processed.
func (s *Store) CompleteWebhookProcessing(ctx context.Context, completion WebhookCompletion) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if completion.NewStatus != "" && completion.OrderID != nil {
		_, err := tx.Exec(ctx, `
			UPDATE orders SET status = $2::order_status,
				stripe_payment_intent_id = COALESCE($3, stripe_payment_intent_id),
				paid_at = COALESCE($4, paid_at),
				updated_at = now()
			WHERE id = $1 AND status::text = ANY($5) AND status::text NOT IN ('paid', 'refunded')`,
			*completion.OrderID, completion.NewStatus, completion.PaymentIntentID, completion.PaidAt, completion.AllowedFrom,
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE webhook_events SET processing_status = $2::webhook_processing_status,
			processed_at = now(), order_id = COALESCE($4, order_id)
		WHERE stripe_event_id = $1 AND processing_status::text = $3`,
		completion.StripeEventID, domain.WebhookStatusProcessed, domain.WebhookStatusProcessing, completion.OrderID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// MarkWebhookProcessed marks a processing webhook event as successfully processed.
func (s *Store) MarkWebhookProcessed(ctx context.Context, stripeEventID string, orderID *string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = $2::webhook_processing_status,
			processed_at = now(), order_id = COALESCE($4, order_id)
		WHERE stripe_event_id = $1 AND processing_status::text = $3`,
		stripeEventID, domain.WebhookStatusProcessed, domain.WebhookStatusProcessing, orderID)
	return err
}

// FailWebhookEvent marks a processing webhook event as failed for Stripe retry.
func (s *Store) FailWebhookEvent(ctx context.Context, stripeEventID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = $2::webhook_processing_status
		WHERE stripe_event_id = $1 AND processing_status::text = $3`,
		stripeEventID, domain.WebhookStatusFailed, domain.WebhookStatusProcessing)
	return err
}

// MarkWebhookIgnored marks an unhandled webhook event as ignored.
func (s *Store) MarkWebhookIgnored(ctx context.Context, stripeEventID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = $2::webhook_processing_status, processed_at = now()
		WHERE stripe_event_id = $1`,
		stripeEventID, domain.WebhookStatusIgnored)
	return err
}

// IsStaleWebhookProcessing reports whether a processing event can be reclaimed.
func IsStaleWebhookProcessing(event *WebhookEvent) bool {
	if event == nil || event.ProcessingStatus != domain.WebhookStatusProcessing {
		return false
	}
	started := event.ProcessingStartedAt
	if started == nil {
		return false
	}
	return time.Since(*started) >= staleWebhookProcessing
}
