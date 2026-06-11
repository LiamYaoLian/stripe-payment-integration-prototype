package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/config"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/handler"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/middleware"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/stripeclient"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	stripe := stripeclient.New(cfg.StripeSecretKey)
	orderSvc := service.NewOrderService(store, stripe, cfg.AppFrontendURL)
	webhookSvc := service.NewWebhookService(store, cfg.StripeWebhookSecret)

	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Idempotency-Key", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", handler.NewHealthHandler(store).ServeHTTP)
	r.Get("/api/products", handler.NewProductsHandler(orderSvc).ServeHTTP)
	r.Post("/api/checkout/sessions", handler.NewCheckoutHandler(orderSvc).ServeHTTP)
	orders := handler.NewOrdersHandler(orderSvc)
	r.Get("/api/orders/{id}", orders.GetByID)
	r.Get("/api/orders/by-session/{sessionId}", orders.GetBySession)
	r.Post("/api/webhooks/stripe", handler.NewWebhooksHandler(webhookSvc).ServeHTTP)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
