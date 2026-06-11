package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLogger logs completed HTTP requests with request ID and duration.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", GetRequestID(r.Context()),
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}
