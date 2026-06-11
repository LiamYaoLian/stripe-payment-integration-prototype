package service

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stripe/stripe-go/v82"
)

func TestParseSessionFromFixture(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}
	var event stripe.Event
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	sess, err := parseSession(event)
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID != "cs_test_completed" {
		t.Fatalf("unexpected session id: %s", sess.ID)
	}
	if sess.Status != stripe.CheckoutSessionStatusComplete {
		t.Fatalf("unexpected status: %s", sess.Status)
	}
	if sess.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		t.Fatalf("unexpected payment status: %s", sess.PaymentStatus)
	}
}
