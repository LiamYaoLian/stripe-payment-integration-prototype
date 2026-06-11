package service

import (
	"strings"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/domain"
)

const (
	maxCheckoutItems      = 20
	maxItemQuantity       = 99
	maxIdempotencyKeyLen  = 128
	maxCustomerEmailLen   = 254
	maxMetadataKeyLen     = 40
)

func validateCheckoutInput(input CreateCheckoutInput) error {
	if !domain.ValidUIModes[input.UIMode] {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "uiMode must be hosted or embedded"}
	}
	if len(input.Items) == 0 {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "items required"}
	}
	if len(input.Items) > maxCheckoutItems {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "too many line items"}
	}
	if input.CustomerEmail != "" && len(input.CustomerEmail) > maxCustomerEmailLen {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "customerEmail too long"}
	}
	if input.CustomerEmail != "" && !strings.Contains(input.CustomerEmail, "@") {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "customerEmail invalid"}
	}

	seenProducts := make(map[string]bool, len(input.Items))
	for _, item := range input.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "productId required"}
		}
		if item.Quantity < 1 || item.Quantity > maxItemQuantity {
			return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "quantity must be between 1 and 99"}
		}
		if seenProducts[item.ProductID] {
			return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "duplicate productId in items"}
		}
		seenProducts[item.ProductID] = true
	}
	return validateMetadata(input.Metadata)
}

func validateIdempotencyKey(key string) error {
	if key == "" {
		return nil
	}
	if len(key) > maxIdempotencyKeyLen {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "Idempotency-Key too long"}
	}
	return nil
}

func validateMetadata(meta map[string]string) error {
	if len(meta) > 50 {
		return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "metadata exceeds 50 keys"}
	}
	for key, value := range meta {
		if len(key) > maxMetadataKeyLen {
			return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "metadata key too long"}
		}
		if len(value) > 500 {
			return &api.AppError{Status: 400, Code: "VALIDATION_ERROR", Message: "metadata value exceeds 500 chars"}
		}
	}
	return nil
}
