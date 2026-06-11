package db

import (
	"encoding/json"
	"fmt"
)

const orderSelectColumns = `
	id, order_number, idempotency_key, status::text, total_amount_cents, currency,
	customer_email, stripe_checkout_session_id, stripe_payment_intent_id,
	stripe_checkout_url, stripe_client_secret, ui_mode::text,
	success_url, cancel_url, return_url, metadata, paid_at, created_at, updated_at,
	access_token_hash`

func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func mergeRequestBodyHash(metadata json.RawMessage, requestBodyHash string) (json.RawMessage, error) {
	if metadata == nil {
		metadata = json.RawMessage(`{}`)
	}
	var metaMap map[string]any
	if err := json.Unmarshal(metadata, &metaMap); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	if metaMap == nil {
		metaMap = map[string]any{}
	}
	metaMap["_request_body_hash"] = requestBodyHash
	bytes, err := json.Marshal(metaMap)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return bytes, nil
}
