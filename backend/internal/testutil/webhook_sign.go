package testutil

import (
	"testing"

	"github.com/stripe/stripe-go/v82/webhook"
)

const TestWebhookSecret = "whsec_test_secret"

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
