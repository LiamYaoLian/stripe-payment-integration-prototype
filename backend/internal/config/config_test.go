package config

import (
	"testing"
)

func TestLoadRequiresStripeSecretKey(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when STRIPE_SECRET_KEY missing")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_x")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "")
	t.Setenv("CORS_ORIGIN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("port %q", cfg.Port)
	}
	if cfg.CORSOrigin != "http://localhost:5173" {
		t.Fatalf("cors %q", cfg.CORSOrigin)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("expected default database url")
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_override")
	t.Setenv("PORT", "9090")
	t.Setenv("CORS_ORIGIN", "http://example.com")
	t.Setenv("APP_FRONTEND_URL", "http://frontend.example.com")
	t.Setenv("DATABASE_URL", "postgresql://custom/db")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_override")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("port %q", cfg.Port)
	}
	if cfg.CORSOrigin != "http://example.com" {
		t.Fatalf("cors %q", cfg.CORSOrigin)
	}
	if cfg.AppFrontendURL != "http://frontend.example.com" {
		t.Fatalf("frontend url %q", cfg.AppFrontendURL)
	}
	if cfg.DatabaseURL != "postgresql://custom/db" {
		t.Fatalf("database url %q", cfg.DatabaseURL)
	}
	if cfg.StripeWebhookSecret != "whsec_override" {
		t.Fatalf("webhook secret %q", cfg.StripeWebhookSecret)
	}
}
