package server

import (
	"net/http"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/handler"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/middleware"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

// RouterDeps holds dependencies required to build the HTTP router.
type RouterDeps struct {
	Health     handler.HealthStore
	Products   *service.ProductService
	Orders     *service.OrderService
	Webhooks   *service.WebhookService
	CORSOrigin string
}

// NewRouter wires middleware and API routes.
func NewRouter(deps RouterDeps) http.Handler {
	healthHandler := handler.NewHealthHandlerFromStore(deps.Health)
	productsHandler := handler.NewProductsHandler(deps.Products)
	checkoutHandler := handler.NewCheckoutHandler(deps.Orders)
	ordersHandler := handler.NewOrdersHandler(deps.Orders)
	webhooksHandler := handler.NewWebhooksHandler(deps.Webhooks)

	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger)
	r.Use(middleware.LimitRequestBody(1 << 20)) // 1 MiB
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{deps.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Idempotency-Key", "X-Order-Token", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)
	r.Get("/health", healthHandler.Ready)

	r.Get("/api/products", productsHandler.ServeHTTP)

	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(20, time.Minute))
		r.Post("/api/checkout/sessions", checkoutHandler.ServeHTTP)
	})

	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(120, time.Minute))
		r.Get("/api/orders/{id}", ordersHandler.GetByID)
		r.Get("/api/orders/by-session/{sessionId}", ordersHandler.GetBySession)
	})

	r.Post("/api/webhooks/stripe", webhooksHandler.ServeHTTP)

	return r
}
