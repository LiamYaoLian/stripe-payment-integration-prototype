package config

import (
	"os"
	"testing"
)

func TestProductionValidationRejectsInsecureDefaults(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_bad")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("CORS_ORIGIN", "http://localhost:5173")
	t.Setenv("APP_FRONTEND_URL", "http://localhost:5173")
	t.Setenv("DATABASE_URL", "postgresql://u:p@db:5432/app?sslmode=disable")
	t.Setenv("AUTH_JWT_SECRET", "short")
	t.Setenv("METRICS_API_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected production validation error")
	}
}

func TestDevelopmentAllowsLocalDefaults(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_ok")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "development" {
		t.Fatalf("env %q", cfg.Env)
	}
	os.Unsetenv("ENV")
}
