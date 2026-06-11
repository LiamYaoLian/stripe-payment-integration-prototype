package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/go-chi/httprate"
)

// RateLimitByIP limits requests per client IP and returns a standard API error envelope.
func RateLimitByIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	retryAfter := strconv.Itoa(int(window.Seconds()))
	return httprate.Limit(
		limit,
		window,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", retryAfter)
			api.WriteError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
		}),
	)
}
