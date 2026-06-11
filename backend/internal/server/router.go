package server

import (
	"net/http"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/handler"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// RouterDeps holds dependencies required to build the HTTP router.
type RouterDeps struct {
	Health   handler.HealthStore
	Products handler.ProductCatalog
	Orders   handler.OrderService
	Webhooks handler.WebhookProcessor
	Auth     handler.UserAuthenticator
	Sessions middleware.SessionLookup
	RateLimits middleware.RateLimitCounter
	CORSOrigin         string
	SecureCookies      bool
	MetricsEnabled     bool
	MetricsAPIKey      string
	TracingServiceName string
}

// NewRouter wires middleware and API routes.
func NewRouter(deps RouterDeps) http.Handler {
	healthHandler := handler.NewHealthHandlerFromStore(deps.Health)
	productsHandler := handler.NewProductsHandler(deps.Products)
	checkoutHandler := handler.NewCheckoutHandler(deps.Orders)
	ordersHandler := handler.NewOrdersHandler(deps.Orders)
	webhooksHandler := handler.NewWebhooksHandler(deps.Webhooks)
	authHandler := handler.NewAuthHandler(deps.Auth, deps.SecureCookies)
	metricsHandler := handler.NewMetricsHandler()

	rateLimit := func(name string, limit int, window time.Duration) func(http.Handler) http.Handler {
		if deps.RateLimits != nil {
			return middleware.RateLimitByIPPostgres(deps.RateLimits, name, limit, window)
		}
		return middleware.RateLimitByIP(limit, window)
	}

	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(middleware.Tracing(deps.TracingServiceName))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.RequestID)
	if deps.MetricsEnabled {
		r.Use(middleware.PrometheusMetrics)
	}
	r.Use(middleware.RequestLogger)
	r.Use(middleware.LimitRequestBody(1 << 20)) // 1 MiB
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{deps.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "Traceparent", "Tracestate", "X-API-Key", "X-Order-Token", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)
	r.Get("/health", healthHandler.Ready)

	if deps.MetricsEnabled {
		r.With(middleware.RequireAPIKey(deps.MetricsAPIKey)).Get("/metrics", metricsHandler.ServeHTTP)
	}

	r.Group(func(r chi.Router) {
		r.Use(chimw.Timeout(60 * time.Second))
		r.Get("/api/products", productsHandler.ServeHTTP)

		r.Group(func(r chi.Router) {
			r.Use(rateLimit("auth-register", 5, time.Minute))
			r.Post("/api/auth/register", authHandler.Register)
		})

		r.Group(func(r chi.Router) {
			r.Use(rateLimit("auth-login", 10, time.Minute))
			r.Post("/api/auth/login", authHandler.Login)
		})

		r.Post("/api/auth/logout", authHandler.Logout)

		r.Group(func(r chi.Router) {
			r.Use(rateLimit("auth-forgot-password", 5, time.Minute))
			r.Post("/api/auth/forgot-password", authHandler.ForgotPassword)
		})

		r.Group(func(r chi.Router) {
			r.Use(rateLimit("auth-reset-password", 10, time.Minute))
			r.Post("/api/auth/reset-password", authHandler.ResetPassword)
		})

		r.Group(func(r chi.Router) {
			r.Use(rateLimit("auth-verify-email", 10, time.Minute))
			r.Post("/api/auth/verify-email", authHandler.VerifyEmail)
		})

		r.Group(func(r chi.Router) {
			r.Use(rateLimit("checkout", 20, time.Minute))
			r.Post("/api/checkout/sessions", checkoutHandler.ServeHTTP)
		})

		r.Group(func(r chi.Router) {
			r.Use(rateLimit("orders-read", 120, time.Minute))
			r.Get("/api/orders/{id}", ordersHandler.GetByID)
			r.Get("/api/orders/by-session/{sessionId}", ordersHandler.GetBySession)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireUserSession(deps.Sessions))
			r.Use(rateLimit("auth-user", 60, time.Minute))
			r.Get("/api/auth/me", authHandler.Me)
			r.Get("/api/orders/mine", ordersHandler.ListMine)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(chimw.Timeout(120 * time.Second))
		r.Use(rateLimit("webhooks", 120, time.Minute))
		r.Post("/api/webhooks/stripe", webhooksHandler.ServeHTTP)
	})

	return r
}
