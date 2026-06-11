package stripeclient

import (
	"testing"

	"github.com/stripe/stripe-go/v82"
)

func TestClientRestoresGlobalAPIKey(t *testing.T) {
	previous := stripe.Key
	stripe.Key = "sk_test_previous"
	t.Cleanup(func() { stripe.Key = previous })

	client := New("sk_test_example", "2026-05-27.dahlia")
	_ = client.ExpireCheckoutSession("cs_invalid")

	if stripe.Key != "sk_test_previous" {
		t.Fatalf("expected restored key %q, got %q", "sk_test_previous", stripe.Key)
	}
}
