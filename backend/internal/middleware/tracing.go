package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Tracing instruments HTTP handlers with OpenTelemetry when serviceName is non-empty.
func Tracing(serviceName string) func(http.Handler) http.Handler {
	if serviceName == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, serviceName, otelhttp.WithFilter(shouldTraceRequest))
	}
}

func shouldTraceRequest(r *http.Request) bool {
	switch r.URL.Path {
	case "/health", "/health/live", "/health/ready", "/metrics":
		return false
	default:
		return true
	}
}
