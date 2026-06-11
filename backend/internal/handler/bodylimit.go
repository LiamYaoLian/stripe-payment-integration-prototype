package handler

import (
	"errors"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
)

func isRequestBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func writeBodyTooLarge(w http.ResponseWriter) {
	api.WriteError(w, http.StatusRequestEntityTooLarge, "VALIDATION_ERROR", "request body too large")
}
