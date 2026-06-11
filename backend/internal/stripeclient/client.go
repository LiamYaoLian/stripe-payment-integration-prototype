package stripeclient

import (
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

type Client struct{}

func New(apiKey string) *Client {
	stripe.Key = apiKey
	return &Client{}
}

func (c *Client) CreateCheckoutSession(params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	return session.New(params)
}

func (c *Client) ExpireCheckoutSession(sessionID string) error {
	_, err := session.Expire(sessionID, nil)
	return err
}
