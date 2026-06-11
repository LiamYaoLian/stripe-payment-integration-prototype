package stripeclient

import (
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

// Client wraps Stripe Checkout Session API calls.
type Client struct {
	apiKey string
}

// New returns a Stripe client configured with the given secret key.
func New(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func (c *Client) setKey() func() {
	previous := stripe.Key
	stripe.Key = c.apiKey
	return func() { stripe.Key = previous }
}

// CreateCheckoutSession creates a Stripe Checkout Session.
func (c *Client) CreateCheckoutSession(params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	defer c.setKey()()
	return session.New(params)
}

// ExpireCheckoutSession expires an open checkout session.
func (c *Client) ExpireCheckoutSession(sessionID string) error {
	defer c.setKey()()
	_, err := session.Expire(sessionID, nil)
	return err
}
