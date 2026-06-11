package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func init() {
	prometheus.MustRegister(collectors.NewBuildInfoCollector())
}

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	WebhookEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "stripe_webhook_events_total",
		Help: "Stripe webhook processing outcomes",
	}, []string{"event_type", "outcome"})

	CheckoutSessionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "checkout_sessions_total",
		Help: "Checkout session creation outcomes",
	}, []string{"outcome"})
)
