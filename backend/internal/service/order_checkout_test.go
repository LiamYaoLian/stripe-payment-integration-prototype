package service

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
	"github.com/stripe/stripe-go/v82"
)

const testProductID = "d4j8k2m9q1p7n3s6"

func newTestOrderService(store *testutil.FakeOrderStore, stripe *testutil.FakeStripe) *OrderService {
	store.Products[testProductID] = db.Product{
		ID: testProductID, Name: "Pro License", UnitAmountCents: 4900, Currency: "usd", Active: true,
	}
	return NewOrderService(store, stripe, "http://localhost:5173")
}

func hostedInput() CreateCheckoutInput {
	return CreateCheckoutInput{
		UIMode: "hosted",
		Items:  []CheckoutItemInput{{ProductID: testProductID, Quantity: 1}},
	}
}

func TestCreateCheckoutSessionSuccessHosted(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	svc := newTestOrderService(store, &testutil.FakeStripe{})
	result, err := svc.CreateCheckoutSession(t.Context(), "idem-1", hostedInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.URL == "" || result.SessionID != "cs_test_fake" {
		t.Fatalf("result %+v", result)
	}
}

func TestCreateCheckoutSessionSuccessEmbedded(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	svc := newTestOrderService(store, &testutil.FakeStripe{})
	input := hostedInput()
	input.UIMode = "embedded"
	result, err := svc.CreateCheckoutSession(t.Context(), "", input)
	if err != nil {
		t.Fatal(err)
	}
	if result.ClientSecret == "" {
		t.Fatalf("result %+v", result)
	}
}

func TestCreateCheckoutSessionValidation(t *testing.T) {
	svc := newTestOrderService(testutil.NewFakeOrderStore(), &testutil.FakeStripe{})
	tests := []struct {
		name  string
		input CreateCheckoutInput
		code  string
	}{
		{
			name:  "invalid ui mode",
			input: CreateCheckoutInput{UIMode: "invalid", Items: []CheckoutItemInput{{ProductID: testProductID, Quantity: 1}}},
			code:  "VALIDATION_ERROR",
		},
		{name: "empty items", input: CreateCheckoutInput{UIMode: "hosted"}, code: "VALIDATION_ERROR"},
		{
			name:  "zero quantity",
			input: CreateCheckoutInput{UIMode: "hosted", Items: []CheckoutItemInput{{ProductID: testProductID, Quantity: 0}}},
			code:  "VALIDATION_ERROR",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateCheckoutSession(t.Context(), "", tc.input)
			assertAppError(t, err, 400, tc.code)
		})
	}
}

func TestCreateCheckoutSessionAfterCanceledCreatesNewOrder(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	hash, _ := canonicalBodyHash(hostedInput())
	idem := "idem-canceled"
	canceled := &db.Order{
		ID: "ord-canceled", OrderNumber: "ORD-CANCELED", UIMode: "hosted", Status: "canceled",
		IdempotencyKey: &idem, Metadata: mustMetaHash(t, hash),
	}
	store.OrdersByIdempotency[idem] = canceled
	store.Orders["ord-canceled"] = canceled

	svc := newTestOrderService(store, &testutil.FakeStripe{})
	result, err := svc.CreateCheckoutSession(t.Context(), idem, hostedInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.OrderID == "ord-canceled" {
		t.Fatal("expected new order, got canceled order replay")
	}
	if _, ok := store.OrdersByIdempotency[idem]; !ok {
		t.Fatal("new order should own idempotency key")
	}
	if store.Orders["ord-canceled"].Status != "canceled" {
		t.Fatal("canceled order should remain for audit")
	}
}

func TestCreateCheckoutSessionProductNotFound(t *testing.T) {
	svc := newTestOrderService(testutil.NewFakeOrderStore(), &testutil.FakeStripe{})
	_, err := svc.CreateCheckoutSession(t.Context(), "", CreateCheckoutInput{
		UIMode: "hosted",
		Items:  []CheckoutItemInput{{ProductID: "missing", Quantity: 1}},
	})
	assertAppError(t, err, 404, "PRODUCT_NOT_FOUND")
}

func TestCreateCheckoutSessionMixedCurrency(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	store.Products[testProductID] = db.Product{ID: testProductID, Name: "USD", UnitAmountCents: 100, Currency: "usd", Active: true}
	store.Products["eur-prod"] = db.Product{ID: "eur-prod", Name: "EUR", UnitAmountCents: 100, Currency: "eur", Active: true}
	svc := NewOrderService(store, &testutil.FakeStripe{}, "http://localhost:5173")
	_, err := svc.CreateCheckoutSession(t.Context(), "", CreateCheckoutInput{
		UIMode: "hosted",
		Items: []CheckoutItemInput{
			{ProductID: testProductID, Quantity: 1},
			{ProductID: "eur-prod", Quantity: 1},
		},
	})
	assertAppError(t, err, 400, "VALIDATION_ERROR")
}

func TestCreateCheckoutSessionIdempotencyReplay(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	hash, _ := canonicalBodyHash(hostedInput())
	sess := "cs_existing"
	url := "https://checkout.stripe.com/existing"
	store.OrdersByIdempotency["idem-replay"] = &db.Order{
		ID: "ord-existing", OrderNumber: "ORD-EXIST", UIMode: "hosted", Status: "pending",
		StripeCheckoutSessionID: &sess, StripeCheckoutURL: &url,
		Metadata: mustMetaHash(t, hash),
	}
	svc := newTestOrderService(store, &testutil.FakeStripe{})
	result, err := svc.CreateCheckoutSession(t.Context(), "idem-replay", hostedInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != sess || result.URL != url {
		t.Fatalf("result %+v", result)
	}
}

func TestCreateCheckoutSessionIdempotencyConflict(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	store.OrdersByIdempotency["idem-conflict"] = &db.Order{
		ID: "ord1", Status: "pending",
		Metadata: mustMetaHash(t, "different-hash"),
	}
	svc := newTestOrderService(store, &testutil.FakeStripe{})
	_, err := svc.CreateCheckoutSession(t.Context(), "idem-conflict", hostedInput())
	assertAppError(t, err, 409, "IDEMPOTENCY_CONFLICT")
}

func TestCreateCheckoutSessionCheckoutInProgress(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	hash, _ := canonicalBodyHash(hostedInput())
	store.OrdersByIdempotency["idem-progress"] = &db.Order{
		ID: "ord1", Status: "pending", Metadata: mustMetaHash(t, hash),
	}
	svc := newTestOrderService(store, &testutil.FakeStripe{})
	_, err := svc.CreateCheckoutSession(t.Context(), "idem-progress", hostedInput())
	assertAppError(t, err, 409, "CHECKOUT_IN_PROGRESS")
}

func TestCreateCheckoutSessionStripeError(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	stripe := &testutil.FakeStripe{Err: testutil.ErrStripeAPI}
	svc := newTestOrderService(store, stripe)
	_, err := svc.CreateCheckoutSession(t.Context(), "", hostedInput())
	assertAppError(t, err, 502, "STRIPE_ERROR")
	if len(store.CancelReasons) != 1 || store.CancelReasons[0] != "stripe_api_error" {
		t.Fatalf("cancel reasons %v", store.CancelReasons)
	}
}

func TestCreateCheckoutSessionPersistFailed(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	store.UpdateSessionErr = errors.New("db write failed")
	stripe := &testutil.FakeStripe{Session: &stripe.CheckoutSession{ID: "cs_persist_fail", URL: "https://x"}}
	svc := newTestOrderService(store, stripe)
	_, err := svc.CreateCheckoutSession(t.Context(), "", hostedInput())
	assertAppError(t, err, 502, "STRIPE_ERROR")
	if !stripe.ExpireCalled {
		t.Fatal("expected session expire")
	}
	if len(store.CancelReasons) != 1 || store.CancelReasons[0] != "persist_failed" {
		t.Fatalf("cancel reasons %v", store.CancelReasons)
	}
}

func TestListProductsAndGetOrder(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	store.Products[testProductID] = db.Product{ID: testProductID, Name: "Pro", UnitAmountCents: 4900, Currency: "usd", Active: true}
	token, _ := testutil.NewOrderAccessToken(t)
	orderFixture := &db.Order{ID: "ord1", OrderNumber: "ORD-1", Status: "paid"}
	testutil.WithAccessToken(orderFixture, token)
	store.Orders["ord1"] = orderFixture

	products, err := NewProductService(store).ListProducts(t.Context())
	if err != nil || len(products) != 1 {
		t.Fatalf("products %v err %v", products, err)
	}
	order, err := NewOrderService(store, nil, "http://localhost:5173").GetOrder(t.Context(), "ord1", token)
	if err != nil || order.ID != "ord1" {
		t.Fatalf("order %v err %v", order, err)
	}
}

func TestGetOrderBySessionFound(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	token, _ := testutil.NewOrderAccessToken(t)
	orderFixture := &db.Order{ID: "ord1", OrderNumber: "ORD-1"}
	testutil.WithAccessToken(orderFixture, token)
	store.OrdersBySession["cs_test_abc"] = orderFixture
	svc := NewOrderService(store, nil, "http://localhost:5173")
	order, err := svc.GetOrderBySession(t.Context(), "cs_test_abc", token)
	if err != nil || order.ID != "ord1" {
		t.Fatalf("order %v err %v", order, err)
	}
}

func mustMetaHash(t *testing.T, hash string) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(map[string]any{"_request_body_hash": hash})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
