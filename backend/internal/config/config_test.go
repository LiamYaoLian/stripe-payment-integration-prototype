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
