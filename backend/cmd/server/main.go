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
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/server"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/stripeclient"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}
	setupLogging(cfg.Env)

	ctx := context.Background()
	shutdownTracing, err := telemetry.Setup(ctx, telemetry.Config{
		ServiceName: cfg.OtelServiceName,
		Endpoint:    cfg.OtelExporterOTLPEndpoint,
		Environment: cfg.Env,
	})
	if err != nil {
		slog.Error("tracing setup failed", "error", err)
		os.Exit(1)
	}

	store, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	go runStaleOrderCleanup(store)

	stripe := stripeclient.New(cfg.StripeSecretKey, cfg.StripeAPIVersion)
	productSvc := service.NewProductService(store)
	orderSvc := service.NewOrderService(store, stripe, cfg.AppFrontendURL)
	webhookSvc := service.NewWebhookService(
		store,
		cfg.StripeWebhookSecret,
		cfg.IgnoreStripeAPIVersionMismatch,
		cfg.StripeAPIVersion,
	)
	authSvc := service.NewAuthService(store, cfg.AuthJWTSecret)

	tracingServiceName := ""
	if cfg.OtelExporterOTLPEndpoint != "" {
		tracingServiceName = cfg.OtelServiceName
	}

	r := server.NewRouter(server.RouterDeps{
		Health:             store,
		Products:           productSvc,
		Orders:             orderSvc,
		Webhooks:           webhookSvc,
		Auth:               authSvc,
		CORSOrigin:         cfg.CORSOrigin,
		AuthJWTSecret:      cfg.AuthJWTSecret,
		MetricsEnabled:     cfg.MetricsEnabled,
		MetricsAPIKey:      cfg.MetricsAPIKey,
		TracingServiceName: tracingServiceName,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("server starting",
			"port", cfg.Port,
			"env", cfg.Env,
			"stripe_api_version", cfg.StripeAPIVersion,
			"metrics_enabled", cfg.MetricsEnabled,
			"tracing_enabled", cfg.OtelExporterOTLPEndpoint != "",
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}
	if err := shutdownTracing(shutdownCtx); err != nil {
		slog.Warn("tracing shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}

func setupLogging(env string) {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(handler))
}

func runStaleOrderCleanup(store *db.Store) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		count, err := store.CancelStalePendingOrders(context.Background(), 15*time.Minute)
		if err != nil {
			slog.Warn("stale order cleanup failed", "error", err)
			continue
		}
		if count > 0 {
			slog.Info("canceled stale pending orders", "count", count)
		}
	}
}
