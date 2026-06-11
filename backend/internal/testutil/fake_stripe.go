package testutil

import (
	"errors"

	"github.com/stripe/stripe-go/v82"
)

type FakeStripe struct {
	Session      *stripe.CheckoutSession
	Err          error
	ExpireCalled bool
}

func (f *FakeStripe) CreateCheckoutSession(_ *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Session != nil {
		return f.Session, nil
	}
	return &stripe.CheckoutSession{
		ID:           "cs_test_fake",
		URL:          "https://checkout.stripe.com/test",
		ClientSecret: "cs_test_secret",
	}, nil
}

func (f *FakeStripe) ExpireCheckoutSession(_ string) error {
	f.ExpireCalled = true
	return nil
}

var ErrStripeAPI = errors.New("stripe api down")
