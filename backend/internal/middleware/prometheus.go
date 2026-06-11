package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/metrics"
)

// PrometheusMetrics records request counts for Prometheus scraping.
func PrometheusMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		metrics.HTTPRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(recorder.status),
		).Inc()
		_ = started
	})
}
