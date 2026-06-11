package testutil

import (
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

// NewOrderAccessToken returns a token and hash pair for order access tests.
func NewOrderAccessToken(t *testing.T) (token string, hash string) {
	t.Helper()
	var err error
	token, hash, err = auth.GenerateOrderAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	return token, hash
}

// WithAccessToken sets AccessTokenHash on an order from a known token.
func WithAccessToken(order *db.Order, token string) {
	hash := auth.HashOrderAccessToken(token)
	order.AccessTokenHash = &hash
}
