package stripeclient

import (
	"sync"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

// Client wraps Stripe Checkout Session API calls.
type Client struct {
	apiKey string
	mu     sync.Mutex
}

// New returns a Stripe client configured with the given secret key.
func New(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func (c *Client) withKey(fn func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	previous := stripe.Key
	stripe.Key = c.apiKey
	defer func() { stripe.Key = previous }()
	return fn()
}

// CreateCheckoutSession creates a Stripe Checkout Session with an idempotency key.
func (c *Client) CreateCheckoutSession(params *stripe.CheckoutSessionParams, idempotencyKey string) (*stripe.CheckoutSession, error) {
	if idempotencyKey != "" {
		params.SetIdempotencyKey(idempotencyKey)
	}
	var result *stripe.CheckoutSession
	var callErr error
	err := c.withKey(func() error {
		result, callErr = session.New(params)
		return callErr
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ExpireCheckoutSession expires an open checkout session.
func (c *Client) ExpireCheckoutSession(sessionID string) error {
	return c.withKey(func() error {
		_, err := session.Expire(sessionID, nil)
		return err
	})
}
