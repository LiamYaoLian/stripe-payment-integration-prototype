package db

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rs/xid"
)

func testDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgresql://stripe:stripe@localhost:5434/stripe_payment?sslmode=disable"
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping DB integration test (run without -short)")
	}
	ctx := context.Background()
	store, err := New(ctx, testDatabaseURL())
	if err != nil {
		t.Skip("postgres not available:", err)
	}
	if err := store.Ping(ctx); err != nil {
		store.Close()
		t.Skip("postgres ping failed:", err)
	}
	return store
}

func deleteTestOrder(ctx context.Context, store *Store, orderID string) {
	_, _ = store.Pool().Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, orderID)
	_, _ = store.Pool().Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
}

func deleteTestWebhookEvent(ctx context.Context, store *Store, stripeEventID string) {
	_, _ = store.Pool().Exec(ctx, `DELETE FROM webhook_events WHERE stripe_event_id = $1`, stripeEventID)
}

func deleteTestProduct(ctx context.Context, store *Store, productID string) {
	_, _ = store.Pool().Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
}

func insertTestProduct(t *testing.T, ctx context.Context, store *Store) Product {
	t.Helper()
	id := "test-prod-" + xid.New().String()
	p := Product{ID: id, Name: "Test Product", UnitAmountCents: 1000, Currency: "usd", Active: true}
	if err := store.UpsertProduct(ctx, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { deleteTestProduct(ctx, store, id) })
	return p
}

func TestIntegrationStoreListActiveProducts(t *testing.T) {
	store := openTestStore(t)
	defer store.Close()
	ctx := context.Background()

	p := insertTestProduct(t, ctx, store)
	products, err := store.ListActiveProducts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, prod := range products {
		if prod.ID == p.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected product %s in list", p.ID)
	}
}

func TestIntegrationStoreOrderLifecycle(t *testing.T) {
	store := openTestStore(t)
	defer store.Close()
	ctx := context.Background()

	product := insertTestProduct(t, ctx, store)
	orderID := xid.New().String()
	orderNumber := "ORD-TEST-" + xid.New().String()[:8]
	itemID := xid.New().String()
	meta, _ := json.Marshal(map[string]string{"source": "test"})
	t.Cleanup(func() { deleteTestOrder(ctx, store, orderID) })

	err := store.CreateOrderWithItems(ctx, CreateOrderParams{
		ID: orderID, OrderNumber: orderNumber, TotalAmountCents: product.UnitAmountCents, Currency: "usd",
		UIMode: "hosted", Metadata: meta, RequestBodyHash: "hash123",
	}, []CreateOrderItemParams{{
		ID: itemID, OrderID: orderID, ProductID: product.ID, ProductName: product.Name,
		Quantity: 1, UnitAmountCents: product.UnitAmountCents, LineTotalCents: product.UnitAmountCents,
	}})
	if err != nil {
		t.Fatal(err)
	}

	sessID := "cs_test_" + xid.New().String()
	url := "https://checkout.stripe.com/test"
	if err := store.UpdateOrderSession(ctx, orderID, sessID, url, ""); err != nil {
		t.Fatal(err)
	}

	bySession, err := store.GetOrderBySessionID(ctx, sessID)
	if err != nil || bySession == nil || bySession.ID != orderID {
		t.Fatalf("bySession %+v err %v", bySession, err)
	}

	byID, err := store.GetOrderByID(ctx, orderID)
	if err != nil || len(byID.Items) != 1 {
		t.Fatalf("byID %+v err %v", byID, err)
	}

	now := time.Now().UTC()
	pi := "pi_test_" + xid.New().String()
	ok, err := store.UpdateOrderStatusIfAllowed(ctx, orderID, "paid", &pi, &now, []string{"pending"})
	if err != nil || !ok {
		t.Fatalf("update ok=%v err=%v", ok, err)
	}

	ok, err = store.UpdateOrderStatusIfAllowed(ctx, orderID, "expired", nil, nil, []string{"pending"})
	if err != nil || ok {
		t.Fatalf("paid order should not revert, ok=%v err=%v", ok, err)
	}
}

func TestIntegrationStoreWebhookEventLifecycle(t *testing.T) {
	store := openTestStore(t)
	defer store.Close()
	ctx := context.Background()

	eventID := "evt_test_" + xid.New().String()
	t.Cleanup(func() { deleteTestWebhookEvent(ctx, store, eventID) })
	payload := json.RawMessage(`{"id":"` + eventID + `"}`)
	if err := store.InsertWebhookEvent(ctx, WebhookEvent{
		ID: xid.New().String(), StripeEventID: eventID, EventType: "charge.succeeded",
		ProcessingStatus: "received", Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}

	claimed, err := store.ClaimWebhookEvent(ctx, eventID)
	if err != nil || !claimed {
		t.Fatalf("claimed=%v err=%v", claimed, err)
	}
	claimed, err = store.ClaimWebhookEvent(ctx, eventID)
	if err != nil || claimed {
		t.Fatalf("second claim claimed=%v err=%v", claimed, err)
	}

	if err := store.MarkWebhookIgnored(ctx, eventID); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationStoreCancelAndClearIdempotency(t *testing.T) {
	store := openTestStore(t)
	defer store.Close()
	ctx := context.Background()

	orderID := xid.New().String()
	idem := "idem-" + xid.New().String()
	t.Cleanup(func() { deleteTestOrder(ctx, store, orderID) })

	err := store.CreateOrderWithItems(ctx, CreateOrderParams{
		ID: orderID, OrderNumber: "ORD-CANCEL-" + xid.New().String()[:6],
		IdempotencyKey: &idem, TotalAmountCents: 500, Currency: "usd", UIMode: "hosted",
		RequestBodyHash: "h",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.CancelOrder(ctx, orderID, "stripe_api_error"); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearOrderIdempotencyKey(ctx, orderID); err != nil {
		t.Fatal(err)
	}
	found, err := store.GetOrderByIdempotencyKey(ctx, idem)
	if err != nil || found != nil {
		t.Fatalf("expected idempotency cleared, found=%+v err=%v", found, err)
	}
}
