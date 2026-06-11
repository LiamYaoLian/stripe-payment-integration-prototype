package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/telemetry"
)

// RequestLogger logs completed HTTP requests with request ID, status, and duration.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"request_id", GetRequestID(r.Context()),
			"duration_ms", time.Since(started).Milliseconds(),
		}
		attrs = append(attrs, telemetry.LogFields(r.Context())...)
		slog.Info("request completed", attrs...)
	})
}
