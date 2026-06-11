package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type canonicalCheckoutBody struct {
	UIMode        string              `json:"uiMode"`
	Items         []CheckoutItemInput `json:"items"`
	CustomerEmail string              `json:"customerEmail,omitempty"`
	Metadata      []canonicalMetadata `json:"metadata,omitempty"`
}

type canonicalMetadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func canonicalBodyHash(input CreateCheckoutInput) (string, error) {
	body := canonicalCheckoutBody{
		UIMode:        input.UIMode,
		Items:         input.Items,
		CustomerEmail: input.CustomerEmail,
	}
	if len(input.Metadata) > 0 {
		keys := make([]string, 0, len(input.Metadata))
		for key := range input.Metadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		body.Metadata = make([]canonicalMetadata, 0, len(keys))
		for _, key := range keys {
			body.Metadata = append(body.Metadata, canonicalMetadata{
				Key: key, Value: input.Metadata[key],
			})
		}
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}
