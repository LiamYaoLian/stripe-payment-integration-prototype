package server

import (
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/handler"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/middleware"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type RouterDeps struct {
	Health     handler.HealthStore
	Orders     *service.OrderService
	Webhooks   *service.WebhookService
	CORSOrigin string
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{deps.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Idempotency-Key", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", handler.NewHealthHandlerFromStore(deps.Health).ServeHTTP)
	r.Get("/api/products", handler.NewProductsHandler(deps.Orders).ServeHTTP)
	r.Post("/api/checkout/sessions", handler.NewCheckoutHandler(deps.Orders).ServeHTTP)
	orders := handler.NewOrdersHandler(deps.Orders)
	r.Get("/api/orders/{id}", orders.GetByID)
	r.Get("/api/orders/by-session/{sessionId}", orders.GetBySession)
	r.Post("/api/webhooks/stripe", handler.NewWebhooksHandler(deps.Webhooks).ServeHTTP)

	return r
}
