package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

var errOrderNotFoundForSession = errors.New("order not found for checkout session")

func webhookStoreError(outcome WebhookOutcome, err error) (WebhookOutcome, error) {
	slog.Error("webhook store error", "error", err, "outcome", outcome)
	return outcome, nil
}

func resolveWebhookOrder(order *db.Order, sessionID string, err error) (*db.Order, error) {
	if err != nil {
		return nil, fmt.Errorf("get order by session %q: %w", sessionID, err)
	}
	if order == nil {
		return nil, fmt.Errorf("%w: %s", errOrderNotFoundForSession, sessionID)
	}
	return order, nil
}
