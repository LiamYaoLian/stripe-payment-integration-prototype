package testutil

import (
	"testing"

	"github.com/stripe/stripe-go/v82/webhook"
)

const (
	TestWebhookSecret    = "whsec_test_secret"
	TestStripeAPIVersion = "2026-05-27.dahlia"
)

func SignWebhookPayload(t *testing.T, payload []byte, secret string) string {
	t.Helper()
	if secret == "" {
		secret = TestWebhookSecret
	}
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload: payload,
		Secret:  secret,
	})
	return signed.Header
}
