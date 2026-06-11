package stripeclient

import (
	"testing"

	"github.com/stripe/stripe-go/v82"
)

func TestClientDoesNotMutateGlobalAPIKey(t *testing.T) {
	previous := stripe.Key
	stripe.Key = "sk_test_previous"
	t.Cleanup(func() { stripe.Key = previous })

	client := New("sk_test_example", "2026-05-27.dahlia")
	_ = client.ExpireCheckoutSession("cs_invalid")

	if stripe.Key != "sk_test_previous" {
		t.Fatalf("expected global key unchanged %q, got %q", "sk_test_previous", stripe.Key)
	}
}

func TestApplyAPIVersionSetsHeader(t *testing.T) {
	client := New("sk_test_example", "2026-05-27.dahlia")
	params := &stripe.CheckoutSessionParams{}
	client.applyAPIVersion(&params.Params)

	if got := params.Headers.Get("Stripe-Version"); got != "2026-05-27.dahlia" {
		t.Fatalf("Stripe-Version header %q", got)
	}
}
