package testutil

import (
	"context"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
)

type FakeWebhookStore struct {
	Events       map[string]*db.WebhookEvent
	Orders       map[string]*db.Order
	Ignored      []string
	Claimed      []string
	StatusUpdate []string
	ClaimOK            bool
	UpdateErr          error
	GetWebhookEventErr error
}

func NewFakeWebhookStore() *FakeWebhookStore {
	return &FakeWebhookStore{
		Events:  make(map[string]*db.WebhookEvent),
		Orders:  make(map[string]*db.Order),
		ClaimOK: true,
	}
}

func (f *FakeWebhookStore) GetWebhookEvent(_ context.Context, stripeEventID string) (*db.WebhookEvent, error) {
	if f.GetWebhookEventErr != nil {
		return nil, f.GetWebhookEventErr
	}
	if event, ok := f.Events[stripeEventID]; ok {
		return event, nil
	}
	return nil, nil
}

func (f *FakeWebhookStore) InsertWebhookEvent(_ context.Context, event db.WebhookEvent) error {
	copy := event
	f.Events[event.StripeEventID] = &copy
	return nil
}

func (f *FakeWebhookStore) MarkWebhookIgnored(_ context.Context, stripeEventID string) error {
	f.Ignored = append(f.Ignored, stripeEventID)
	if event, ok := f.Events[stripeEventID]; ok {
		event.ProcessingStatus = domain.WebhookStatusIgnored
	}
	return nil
}

func (f *FakeWebhookStore) ClaimWebhookEvent(_ context.Context, stripeEventID string) (bool, error) {
	f.Claimed = append(f.Claimed, stripeEventID)
	if !f.ClaimOK {
		return false, nil
	}
	if event, ok := f.Events[stripeEventID]; ok {
		event.ProcessingStatus = domain.WebhookStatusProcessing
	}
	return true, nil
}

func (f *FakeWebhookStore) MarkWebhookProcessed(_ context.Context, stripeEventID string, orderID *string) error {
	if event, ok := f.Events[stripeEventID]; ok {
		event.ProcessingStatus = domain.WebhookStatusProcessed
		event.OrderID = orderID
	}
	return nil
}

func (f *FakeWebhookStore) FailWebhookEvent(_ context.Context, stripeEventID string) error {
	if event, ok := f.Events[stripeEventID]; ok {
		event.ProcessingStatus = domain.WebhookStatusFailed
	}
	return nil
}

func (f *FakeWebhookStore) GetOrderBySessionID(_ context.Context, sessionID string) (*db.Order, error) {
	if order, ok := f.Orders[sessionID]; ok {
		return order, nil
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
