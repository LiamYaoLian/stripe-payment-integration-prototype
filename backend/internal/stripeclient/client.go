package stripeclient

import (
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

// Client wraps Stripe Checkout Session API calls with a dedicated backend and API key.
type Client struct {
	checkout   session.Client
	apiVersion string
}

// New returns a Stripe client that does not mutate the global stripe.Key.
func New(apiKey, apiVersion string) *Client {
	backends := stripe.NewBackends(nil)
	return &Client{
		checkout:   session.Client{B: backends.API, Key: apiKey},
		apiVersion: apiVersion,
	}
}

func (c *Client) applyAPIVersion(params *stripe.Params) {
	if c.apiVersion == "" || params == nil {
		return
	}
	if params.Headers == nil {
		params.Headers = make(http.Header)
	}
	params.Headers.Set("Stripe-Version", c.apiVersion)
}

// CreateCheckoutSession creates a Stripe Checkout Session with an idempotency key.
func (c *Client) CreateCheckoutSession(params *stripe.CheckoutSessionParams, idempotencyKey string) (*stripe.CheckoutSession, error) {
	if idempotencyKey != "" {
		params.SetIdempotencyKey(idempotencyKey)
	}
	c.applyAPIVersion(&params.Params)
	return c.checkout.New(params)
}

// ExpireCheckoutSession expires an open checkout session.
func (c *Client) ExpireCheckoutSession(sessionID string) error {
	params := &stripe.CheckoutSessionExpireParams{}
	c.applyAPIVersion(&params.Params)
	_, err := c.checkout.Expire(sessionID, params)
	return err
}
