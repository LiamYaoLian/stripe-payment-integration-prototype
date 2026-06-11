package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/middleware"
)

const internalErrorMessage = "internal server error"

// writeServiceError maps service errors to HTTP responses.
// Unexpected errors are logged and return a generic message to the client.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *api.AppError
	if errors.As(err, &appErr) {
		api.WriteAppError(w, appErr)
		return
	}
	if err != nil {
		slog.Error("request failed",
			"error", err,
			"request_id", middleware.GetRequestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
		)
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", internalErrorMessage)
	}
}
