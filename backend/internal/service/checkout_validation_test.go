package service

import (
	"strings"
	"testing"
)

func TestValidateCheckoutInputRejectsDuplicateProducts(t *testing.T) {
	err := validateCheckoutInput(CreateCheckoutInput{
		UIMode: "hosted",
		Items: []CheckoutItemInput{
			{ProductID: "p1", Quantity: 1},
			{ProductID: "p1", Quantity: 2},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate product error")
	}
}

func TestValidateCheckoutInputRejectsEmptyProductID(t *testing.T) {
	err := validateCheckoutInput(CreateCheckoutInput{
		UIMode: "hosted",
		Items:  []CheckoutItemInput{{ProductID: "  ", Quantity: 1}},
	})
	if err == nil {
		t.Fatal("expected empty productId error")
	}
}

func TestValidateIdempotencyKeyTooLong(t *testing.T) {
	err := validateIdempotencyKey(strings.Repeat("a", 129))
	if err == nil {
		t.Fatal("expected idempotency key error")
	}
}
