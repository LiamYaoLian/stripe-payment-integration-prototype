// Package domain holds shared domain constants and value types.
package domain

// Order status values stored in the database.
const (
	OrderStatusPending    = "pending"
	OrderStatusProcessing = "processing"
	OrderStatusPaid       = "paid"
	OrderStatusFailed     = "failed"
	OrderStatusExpired    = "expired"
	OrderStatusCanceled   = "canceled"
)

// Webhook processing status values.
const (
	WebhookStatusReceived  = "received"
	WebhookStatusProcessed = "processed"
	WebhookStatusIgnored   = "ignored"
	WebhookStatusFailed    = "failed"
)
