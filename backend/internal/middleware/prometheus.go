package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/metrics"
	"github.com/go-chi/chi/v5"
)

// PrometheusMetrics records request counts and latency for Prometheus scraping.
func PrometheusMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		path := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil {
			if pattern := rc.RoutePattern(); pattern != "" {
				path = pattern
			}
		}

		elapsed := time.Since(started).Seconds()
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(recorder.status)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(elapsed)
	})
}
