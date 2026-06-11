package service

// WebhookOutcome describes the result of processing a Stripe webhook without HTTP details.
type WebhookOutcome int

const (
	// WebhookOutcomeInvalidSignature means signature verification failed.
	WebhookOutcomeInvalidSignature WebhookOutcome = iota
	// WebhookOutcomeAcknowledged means Stripe should treat the event as received.
	WebhookOutcomeAcknowledged
	// WebhookOutcomeRetryLater means Stripe should retry the event later.
	WebhookOutcomeRetryLater
	// WebhookOutcomeProcessingFailed means processing failed and Stripe should retry.
	WebhookOutcomeProcessingFailed
)
