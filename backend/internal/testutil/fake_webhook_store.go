package testutil

import (
	"context"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

type FakeWebhookStore struct {
	Events       map[string]*db.WebhookEvent
	Orders       map[string]*db.Order
	Ignored      []string
	Claimed      []string
	StatusUpdate []string
	ClaimOK      bool
	UpdateErr    error
}

func NewFakeWebhookStore() *FakeWebhookStore {
	return &FakeWebhookStore{
		Events:  make(map[string]*db.WebhookEvent),
		Orders:  make(map[string]*db.Order),
		ClaimOK: true,
	}
}

func (f *FakeWebhookStore) GetWebhookEvent(_ context.Context, stripeEventID string) (*db.WebhookEvent, error) {
	if e, ok := f.Events[stripeEventID]; ok {
		return e, nil
	}
	return nil, nil
}

func (f *FakeWebhookStore) InsertWebhookEvent(_ context.Context, e db.WebhookEvent) error {
	copy := e
	f.Events[e.StripeEventID] = &copy
	return nil
}

func (f *FakeWebhookStore) MarkWebhookIgnored(_ context.Context, stripeEventID string) error {
	f.Ignored = append(f.Ignored, stripeEventID)
	if e, ok := f.Events[stripeEventID]; ok {
		e.ProcessingStatus = "ignored"
	}
	return nil
}

func (f *FakeWebhookStore) ClaimWebhookEvent(_ context.Context, stripeEventID string) (bool, error) {
	f.Claimed = append(f.Claimed, stripeEventID)
	if !f.ClaimOK {
		return false, nil
	}
	if e, ok := f.Events[stripeEventID]; ok {
		e.ProcessingStatus = "processed"
	}
	return true, nil
}

func (f *FakeWebhookStore) FailWebhookEvent(_ context.Context, stripeEventID string) error {
	if e, ok := f.Events[stripeEventID]; ok {
		e.ProcessingStatus = "failed"
	}
	return nil
}

func (f *FakeWebhookStore) GetOrderBySessionID(_ context.Context, sessionID string) (*db.Order, error) {
	if o, ok := f.Orders[sessionID]; ok {
		return o, nil
	}
	return nil, nil
}

func (f *FakeWebhookStore) UpdateOrderStatusIfAllowed(_ context.Context, orderID, newStatus string, _ *string, _ *time.Time, _ []string) (bool, error) {
	if f.UpdateErr != nil {
		return false, f.UpdateErr
	}
	f.StatusUpdate = append(f.StatusUpdate, orderID+":"+newStatus)
	return true, nil
}
