package stripeclient

import (
	"testing"

	"github.com/stripe/stripe-go/v82"
)

func TestNewSetsAPIKey(t *testing.T) {
	prev := stripe.Key
	t.Cleanup(func() { stripe.Key = prev })

	New("sk_test_example")
	if stripe.Key != "sk_test_example" {
		t.Fatalf("key %q", stripe.Key)
	}
}
