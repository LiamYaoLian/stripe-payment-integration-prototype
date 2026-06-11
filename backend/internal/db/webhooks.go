package db

import (
	"context"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

// GetWebhookEvent returns a webhook event by Stripe event ID.
func (s *Store) GetWebhookEvent(ctx context.Context, stripeEventID string) (*WebhookEvent, error) {
	var event WebhookEvent
	err := s.pool.QueryRow(ctx, `
		SELECT id, stripe_event_id, event_type, order_id, processing_status::text, payload, processed_at
		FROM webhook_events WHERE stripe_event_id = $1`, stripeEventID).
		Scan(&event.ID, &event.StripeEventID, &event.EventType, &event.OrderID, &event.ProcessingStatus, &event.Payload, &event.ProcessedAt)
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

// ClaimWebhookEvent moves a webhook event from received/failed to processing.
func (s *Store) ClaimWebhookEvent(ctx context.Context, stripeEventID string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = $2::webhook_processing_status
		WHERE stripe_event_id = $1 AND processing_status::text = ANY($3)`,
		stripeEventID, domain.WebhookStatusProcessing,
		[]string{domain.WebhookStatusReceived, domain.WebhookStatusFailed})
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// MarkWebhookProcessed marks a processing webhook event as successfully processed.
func (s *Store) MarkWebhookProcessed(ctx context.Context, stripeEventID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = $2::webhook_processing_status, processed_at = now()
		WHERE stripe_event_id = $1 AND processing_status::text = $3`,
		stripeEventID, domain.WebhookStatusProcessed, domain.WebhookStatusProcessing)
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
