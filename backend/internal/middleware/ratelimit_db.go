package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
)

// RateLimitCounter increments distributed rate-limit buckets.
type RateLimitCounter interface {
	IncrementRateLimitBucket(ctx context.Context, bucketKey string, windowStart time.Time) (int, error)
}

// RateLimitByIPPostgres limits requests per client IP using a shared Postgres counter.
func RateLimitByIPPostgres(store RateLimitCounter, name string, limit int, window time.Duration) func(http.Handler) http.Handler {
	retryAfter := strconv.Itoa(int(window.Seconds()))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			windowStart := time.Now().UTC().Truncate(window)
			clientIP := r.RemoteAddr
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				clientIP = host
			}
			bucketKey := fmt.Sprintf("%s:%s", name, clientIP)
			count, err := store.IncrementRateLimitBucket(r.Context(), bucketKey, windowStart)
			if err != nil {
				api.WriteError(w, http.StatusServiceUnavailable, "RATE_LIMIT_UNAVAILABLE", "rate limit check failed")
				return
			}
			if count > limit {
				w.Header().Set("Retry-After", retryAfter)
				api.WriteError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
