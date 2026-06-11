package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/rs/xid"
)

func TestReplayCheckoutHosted(t *testing.T) {
	url := "https://checkout.stripe.com/c/pay/cs_test_abc"
	sess := "cs_test_abc"
	o := &db.Order{
		ID: "ord1", OrderNumber: "ORD-001", UIMode: "hosted",
		StripeCheckoutSessionID: &sess, StripeCheckoutURL: &url,
	}
	r := replayCheckout(o)
	if r.OrderID != "ord1" || r.SessionID != sess || r.URL != url {
		t.Fatalf("unexpected replay: %+v", r)
	}
	if r.ClientSecret != "" {
		t.Fatal("hosted replay should not include client secret")
	}
}

func TestReplayCheckoutEmbedded(t *testing.T) {
	secret := "cs_test_secret"
	sess := "cs_test_xyz"
	o := &db.Order{
		ID: "ord2", OrderNumber: "ORD-002", UIMode: "embedded",
		StripeCheckoutSessionID: &sess, StripeClientSecret: &secret,
	}
	r := replayCheckout(o)
	if r.ClientSecret != secret {
		t.Fatalf("expected client secret, got %+v", r)
	}
	if r.URL != "" {
		t.Fatal("embedded replay should not include url")
	}
}

func TestValidateMetadata(t *testing.T) {
	big := make(map[string]string, 51)
	for i := 0; i < 51; i++ {
		big[fmt.Sprintf("k%d", i)] = "v"
	}
	if err := validateMetadata(big); err == nil {
		t.Fatal("expected error for >50 keys")
	}

	long := map[string]string{"k": string(make([]byte, 501))}
	if err := validateMetadata(long); err == nil {
		t.Fatal("expected error for long value")
	}
}

func TestCanonicalBodyHashStable(t *testing.T) {
	input := CreateCheckoutInput{
		UIMode: "hosted",
		Items:  []CheckoutItemInput{{ProductID: "p1", Quantity: 1}},
	}
	h1, err := canonicalBodyHash(input)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := canonicalBodyHash(input)
	if err != nil || h1 != h2 {
		t.Fatalf("hash not stable: %s vs %s", h1, h2)
	}
}

func TestExtractRequestHash(t *testing.T) {
	meta, _ := json.Marshal(map[string]any{"_request_body_hash": "abc123"})
	if got := extractRequestHash(meta); got != "abc123" {
		t.Fatalf("got %q", got)
	}
	if got := extractRequestHash(json.RawMessage(`not-json`)); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestGenerateOrderNumberUnique(t *testing.T) {
	seen := make(map[string]bool, 100)
	for range 100 {
		id := xid.New().String()
		n := generateOrderNumber(id)
		if seen[n] {
			t.Fatalf("duplicate order number: %s", n)
		}
		seen[n] = true
		if !strings.HasPrefix(n, "ORD-") {
			t.Fatalf("unexpected format: %s", n)
		}
	}
}

func TestCreateCheckoutSessionInvalidUIMode(t *testing.T) {
	svc := NewOrderService(nil, nil, "http://localhost:5173")
	_, err := svc.CreateCheckoutSession(t.Context(), "", CreateCheckoutInput{
		UIMode: "invalid",
		Items:  []CheckoutItemInput{{ProductID: "p1", Quantity: 1}},
	})
	assertAppError(t, err, 400, "VALIDATION_ERROR")
}

func TestCreateCheckoutSessionEmptyItems(t *testing.T) {
	svc := NewOrderService(nil, nil, "http://localhost:5173")
	_, err := svc.CreateCheckoutSession(t.Context(), "", CreateCheckoutInput{UIMode: "hosted"})
	assertAppError(t, err, 400, "VALIDATION_ERROR")
}

func assertAppError(t *testing.T, err error, status int, code string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*api.AppError)
	if !ok || appErr.Status != status || appErr.Code != code {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetOrderBySessionInvalidPrefix(t *testing.T) {
	svc := NewOrderService(nil, nil, "http://localhost:5173")
	_, err := svc.GetOrderBySession(t.Context(), "pi_invalid")
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*api.AppError)
	if !ok || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("unexpected error: %v", err)
	}
}
