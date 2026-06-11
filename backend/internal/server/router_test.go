package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
)

func newTestRouter(t *testing.T, health testutil.FakeHealth) (*testutil.FakeOrderStore, http.Handler) {
	t.Helper()
	store := testutil.NewFakeOrderStore()
	store.Products["p1"] = db.Product{ID: "p1", Name: "Pro", UnitAmountCents: 4900, Currency: "usd", Active: true}
	products := service.NewProductService(store)
	orders := service.NewOrderService(store, &testutil.FakeStripe{}, "http://localhost:5173")
	webhooks := service.NewWebhookService(testutil.NewFakeWebhookStore(), testutil.TestWebhookSecret, true, testutil.TestStripeAPIVersion)
	authSvc := service.NewAuthService(testutil.NewFakeCustomerStore(), "test-jwt-secret")
	return store, NewRouter(RouterDeps{
		Health:         health,
		Products:       products,
		Orders:         orders,
		Webhooks:       webhooks,
		Auth:           authSvc,
		CORSOrigin:     "http://localhost:5173",
		AuthJWTSecret:  "test-jwt-secret",
		MetricsEnabled: false,
	})
}

func TestRouterHealthOK(t *testing.T) {
	_, r := newTestRouter(t, testutil.FakeHealth{})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data["status"] != "ok" || env.Data["db"] != "connected" {
		t.Fatalf("data %+v", env.Data)
	}
}

func TestRouterHealthDBDisconnected(t *testing.T) {
	_, r := newTestRouter(t, testutil.FakeHealth{Err: os.ErrDeadlineExceeded})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data map[string]string `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&env)
	if env.Data["db"] != "disconnected" {
		t.Fatalf("data %+v", env.Data)
	}
}

func TestRouterListProducts(t *testing.T) {
	_, r := newTestRouter(t, testutil.FakeHealth{})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/products", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouterCreateCheckoutSession(t *testing.T) {
	_, r := newTestRouter(t, testutil.FakeHealth{})
	body, _ := json.Marshal(service.CreateCheckoutInput{
		UIMode: "hosted",
		Items:  []service.CheckoutItemInput{{ProductID: "p1", Quantity: 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/checkout/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "router-test-key")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouterGetOrderBySession(t *testing.T) {
	store, r := newTestRouter(t, testutil.FakeHealth{})
	sessID := "cs_router_test"
	token, _ := testutil.NewOrderAccessToken(t)
	order := &db.Order{
		ID: "ord-router", OrderNumber: "ORD-R", Status: "paid",
		TotalAmountCents: 4900, Currency: "usd",
	}
	testutil.WithAccessToken(order, token)
	store.OrdersBySession[sessID] = order
	req := httptest.NewRequest(http.MethodGet, "/api/orders/by-session/"+sessID, nil)
	req.Header.Set("X-Order-Token", token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouterWebhookCheckoutCompleted(t *testing.T) {
	payload, err := os.ReadFile("../handler/testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeOrderStore()
	whStore := testutil.NewFakeWebhookStore()
	whStore.Orders["cs_test_completed"] = &db.Order{ID: "ord-wh", Status: "pending", TotalAmountCents: 4900, Currency: "usd"}
	products := service.NewProductService(store)
	orders := service.NewOrderService(store, &testutil.FakeStripe{}, "http://localhost:5173")
	webhooks := service.NewWebhookService(whStore, testutil.TestWebhookSecret, true, testutil.TestStripeAPIVersion)
	authSvc := service.NewAuthService(testutil.NewFakeCustomerStore(), "test-jwt-secret")
	r := NewRouter(RouterDeps{
		Health: testutil.FakeHealth{}, Products: products, Orders: orders, Webhooks: webhooks, Auth: authSvc,
		CORSOrigin: "http://localhost:5173", AuthJWTSecret: "test-jwt-secret", MetricsEnabled: false,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", testutil.SignWebhookPayload(t, payload, ""))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouterCORSOptions(t *testing.T) {
	_, r := newTestRouter(t, testutil.FakeHealth{})
	req := httptest.NewRequest(http.MethodOptions, "/api/products", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS headers")
	}
}
