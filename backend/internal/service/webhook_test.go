package service

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
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

func TestPaymentIntentID(t *testing.T) {
	if paymentIntentID(&stripe.CheckoutSession{}) != nil {
		t.Fatal("expected nil")
	}
	sess := &stripe.CheckoutSession{PaymentIntent: &stripe.PaymentIntent{ID: "pi_123"}}
	got := paymentIntentID(sess)
	if got == nil || *got != "pi_123" {
		t.Fatalf("got %v", got)
	}
}

func TestWebhookHandleInvalidSignature(t *testing.T) {
	svc := NewWebhookService(testutil.NewFakeWebhookStore(), testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), []byte(`{}`), "bad-sig")
	if err != nil || result.StatusCode != 400 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestWebhookHandleIgnoredEvent(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/charge.succeeded.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), raw, testutil.SignWebhookPayload(t, raw, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(store.Ignored) != 1 {
		t.Fatalf("ignored %v", store.Ignored)
	}
}

func TestWebhookHandleAlreadyProcessed(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/charge.succeeded.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	store.Events["evt_test_charge"] = &db.WebhookEvent{
		StripeEventID: "evt_test_charge", ProcessingStatus: "processed",
	}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), raw, testutil.SignWebhookPayload(t, raw, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestWebhookHandleInFlight(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/charge.succeeded.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	store.Events["evt_test_charge"] = &db.WebhookEvent{
		StripeEventID: "evt_test_charge", ProcessingStatus: "received",
	}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), raw, testutil.SignWebhookPayload(t, raw, ""))
	if err != nil || result.StatusCode != 503 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestWebhookHandleCheckoutCompletedPaid(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_completed"] = &db.Order{ID: "ord1", Status: "pending"}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), raw, testutil.SignWebhookPayload(t, raw, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(store.StatusUpdate) != 1 || store.StatusUpdate[0] != "ord1:paid" {
		t.Fatalf("updates %v", store.StatusUpdate)
	}
}

func TestWebhookHandleSessionExpired(t *testing.T) {
	payload := []byte(`{
		"id": "evt_expired",
		"object": "event",
		"type": "checkout.session.expired",
		"api_version": "2026-05-27.dahlia",
		"data": {"object": {"id": "cs_test_expired", "object": "checkout.session", "status": "expired"}}
	}`)
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_expired"] = &db.Order{ID: "ord2", Status: "pending"}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), payload, testutil.SignWebhookPayload(t, payload, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(store.StatusUpdate) != 1 || store.StatusUpdate[0] != "ord2:expired" {
		t.Fatalf("updates %v", store.StatusUpdate)
	}
}

func TestWebhookHandleCheckoutCompletedIncompleteSession(t *testing.T) {
	payload := []byte(`{
		"id": "evt_open",
		"object": "event",
		"type": "checkout.session.completed",
		"api_version": "2026-05-27.dahlia",
		"data": {"object": {
			"id": "cs_test_open",
			"object": "checkout.session",
			"status": "open",
			"payment_status": "unpaid"
		}}
	}`)
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_open"] = &db.Order{ID: "ord-open", Status: "pending"}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), payload, testutil.SignWebhookPayload(t, payload, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(store.StatusUpdate) != 0 {
		t.Fatalf("expected no status updates for open session, got %v", store.StatusUpdate)
	}
}

func TestWebhookHandleCheckoutCompletedProcessing(t *testing.T) {
	payload := []byte(`{
		"id": "evt_processing",
		"object": "event",
		"type": "checkout.session.completed",
		"api_version": "2026-05-27.dahlia",
		"data": {"object": {
			"id": "cs_test_processing",
			"object": "checkout.session",
			"status": "complete",
			"payment_status": "unpaid",
			"payment_intent": "pi_async"
		}}
	}`)
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_processing"] = &db.Order{ID: "ord4", Status: "pending"}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), payload, testutil.SignWebhookPayload(t, payload, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(store.StatusUpdate) != 1 || store.StatusUpdate[0] != "ord4:processing" {
		t.Fatalf("updates %v", store.StatusUpdate)
	}
}

func TestWebhookHandleAsyncSucceeded(t *testing.T) {
	payload := []byte(`{
		"id": "evt_async_ok",
		"object": "event",
		"type": "checkout.session.async_payment_succeeded",
		"api_version": "2026-05-27.dahlia",
		"data": {"object": {
			"id": "cs_test_async",
			"object": "checkout.session",
			"status": "complete",
			"payment_intent": {"id": "pi_async"}
		}}
	}`)
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_async"] = &db.Order{ID: "ord_async", Status: "processing"}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), payload, testutil.SignWebhookPayload(t, payload, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(store.StatusUpdate) != 1 || store.StatusUpdate[0] != "ord_async:paid" {
		t.Fatalf("updates %v", store.StatusUpdate)
	}
}

func TestWebhookHandleClaimRaceReturns503(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	store.Events["evt_test_completed"] = &db.WebhookEvent{
		StripeEventID: "evt_test_completed", ProcessingStatus: "received",
	}
	store.ClaimOK = false
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), raw, testutil.SignWebhookPayload(t, raw, ""))
	if err != nil || result.StatusCode != 503 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestWebhookHandleProcessErrorReturns500(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_completed"] = &db.Order{ID: "ord1", Status: "pending"}
	store.UpdateErr = errors.New("db update failed")
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), raw, testutil.SignWebhookPayload(t, raw, ""))
	if err != nil || result.StatusCode != 500 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestWebhookHandleFailedEventRetryReturns500(t *testing.T) {
	raw, err := os.ReadFile("../handler/testdata/checkout.session.completed.json")
	if err != nil {
		t.Fatal(err)
	}
	store := testutil.NewFakeWebhookStore()
	store.Events["evt_test_completed"] = &db.WebhookEvent{
		StripeEventID: "evt_test_completed", ProcessingStatus: "failed",
	}
	store.ClaimOK = false
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), raw, testutil.SignWebhookPayload(t, raw, ""))
	if err != nil || result.StatusCode != 500 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestWebhookHandleAsyncPaymentFailed(t *testing.T) {
	payload := []byte(`{
		"id": "evt_failed",
		"object": "event",
		"type": "checkout.session.async_payment_failed",
		"api_version": "2026-05-27.dahlia",
		"data": {"object": {"id": "cs_test_failed", "object": "checkout.session", "status": "complete"}}
	}`)
	store := testutil.NewFakeWebhookStore()
	store.Orders["cs_test_failed"] = &db.Order{ID: "ord3", Status: "processing"}
	svc := NewWebhookService(store, testutil.TestWebhookSecret)
	result, err := svc.Handle(t.Context(), payload, testutil.SignWebhookPayload(t, payload, ""))
	if err != nil || result.StatusCode != 200 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(store.StatusUpdate) != 1 || store.StatusUpdate[0] != "ord3:failed" {
		t.Fatalf("updates %v", store.StatusUpdate)
	}
}
