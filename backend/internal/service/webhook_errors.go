package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/stripe/stripe-go/v82"
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

func verifyCheckoutSessionAmount(order *db.Order, session *stripe.CheckoutSession) error {
	if session.AmountTotal <= 0 {
		return fmt.Errorf("checkout session %s missing amount_total", session.ID)
	}
	if int64(order.TotalAmountCents) != session.AmountTotal {
		return fmt.Errorf("session amount mismatch: order=%d session=%d", order.TotalAmountCents, session.AmountTotal)
	}
	if session.Currency == "" {
		return fmt.Errorf("checkout session %s missing currency", session.ID)
	}
	if !strings.EqualFold(string(session.Currency), order.Currency) {
		return fmt.Errorf("session currency mismatch: order=%s session=%s", order.Currency, session.Currency)
	}
	return nil
}

func verifyChargeAmount(order *db.Order, charge *stripe.Charge) error {
	if charge.Amount <= 0 {
		return fmt.Errorf("charge %s missing amount", charge.ID)
	}
	if int64(order.TotalAmountCents) != charge.Amount {
		return fmt.Errorf("charge amount mismatch: order=%d charge=%d", order.TotalAmountCents, charge.Amount)
	}
	if charge.Currency == "" {
		return fmt.Errorf("charge %s missing currency", charge.ID)
	}
	if !strings.EqualFold(string(charge.Currency), order.Currency) {
		return fmt.Errorf("charge currency mismatch: order=%s charge=%s", order.Currency, charge.Currency)
	}
	return nil
}
