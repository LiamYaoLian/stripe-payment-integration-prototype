package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
	"github.com/rs/xid"
)

func openE2EStore(t *testing.T) *db.Store {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping e2e webhook test (run without -short)")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgresql://stripe:stripe@localhost:5434/stripe_payment?sslmode=disable"
	}
	store, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Skip("postgres not available:", err)
	}
	return store
}

func TestE2EWebhookCheckoutCompleted(t *testing.T) {
	store := openE2EStore(t)
	defer store.Close()
	ctx := context.Background()

	orderID := xid.New().String()
	sessID := "cs_e2e_" + xid.New().String()[:12]
	eventID := "evt_e2e_" + xid.New().String()[:12]
	t.Cleanup(func() {
		_, _ = store.Pool().Exec(ctx, `DELETE FROM webhook_events WHERE stripe_event_id = $1`, eventID)
		_, _ = store.Pool().Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
	})

	err := store.CreateOrderWithItems(ctx, db.CreateOrderParams{
		ID: orderID, OrderNumber: "ORD-E2E-" + xid.New().String()[:6],
		TotalAmountCents: 1000, Currency: "usd", UIMode: "hosted", RequestBodyHash: "e2e",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateOrderSession(ctx, orderID, sessID, "https://checkout.stripe.com/e2e", ""); err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{
		"id": "` + eventID + `",
		"object": "event",
		"type": "checkout.session.completed",
		"api_version": "2026-05-27.dahlia",
		"data": {"object": {
			"id": "` + sessID + `",
			"object": "checkout.session",
			"status": "complete",
			"payment_status": "paid",
			"amount_total": 1000,
			"currency": "usd",
			"payment_intent": "pi_e2e_` + xid.New().String()[:8] + `"
		}}
	}`)

	products := service.NewProductService(store)
	orders := service.NewOrderService(store, &testutil.FakeStripe{}, "http://localhost:5173")
	webhooks := service.NewWebhookService(store, testutil.TestWebhookSecret, true, testutil.TestStripeAPIVersion)
	authSvc := service.NewAuthService(store, "e2e-jwt-secret")
	router := NewRouter(RouterDeps{
		Health: store, Products: products, Orders: orders, Webhooks: webhooks, Auth: authSvc,
		CORSOrigin: "http://localhost:5173", AuthJWTSecret: "e2e-jwt-secret", MetricsEnabled: false,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", testutil.SignWebhookPayload(t, payload, ""))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}

	order, err := store.GetOrderByID(ctx, orderID)
	if err != nil || order.Status != "paid" {
		t.Fatalf("order status %q err=%v", order.Status, err)
	}
}
