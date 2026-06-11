package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const defaultStripeAPIVersion = "2026-05-27.dahlia"

// Config holds application configuration loaded from environment variables.
type Config struct {
	Env                            string
	Port                           string
	CORSOrigin                     string
	AppFrontendURL                 string
	StripeSecretKey                string
	StripeWebhookSecret            string
	StripeAPIVersion               string
	DatabaseURL                    string
	AuthJWTSecret                  string
	MetricsEnabled                 bool
	MetricsAPIKey                  string
	IgnoreStripeAPIVersionMismatch bool
}

// Load reads configuration from the environment.
func Load() (*Config, error) {
	env := getEnv("ENV", "development")
	if env == "development" {
		_ = godotenv.Load()
	}

	cfg := &Config{
		Env:                            env,
		Port:                           getEnv("PORT", "8080"),
		CORSOrigin:                     getEnv("CORS_ORIGIN", "http://localhost:5173"),
		AppFrontendURL:                 getEnv("APP_FRONTEND_URL", "http://localhost:5173"),
		StripeSecretKey:                os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:            os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeAPIVersion:               getEnv("STRIPE_API_VERSION", defaultStripeAPIVersion),
		DatabaseURL:                    getEnv("DATABASE_URL", "postgresql://stripe:stripe@localhost:5434/stripe_payment?sslmode=disable"),
		AuthJWTSecret:                  os.Getenv("AUTH_JWT_SECRET"),
		MetricsEnabled:                 getEnv("METRICS_ENABLED", "true") == "true",
		MetricsAPIKey:                  os.Getenv("METRICS_API_KEY"),
		IgnoreStripeAPIVersionMismatch: env != "production",
	}

	if cfg.StripeSecretKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is required")
	}
	if cfg.StripeWebhookSecret == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET is required")
	}
	if cfg.AuthJWTSecret == "" && env == "development" {
		cfg.AuthJWTSecret = "dev-only-change-me"
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Env != "production" {
		return nil
	}
	if strings.Contains(c.CORSOrigin, "localhost") {
		return fmt.Errorf("CORS_ORIGIN must not use localhost when ENV=production")
	}
	if !strings.HasPrefix(c.AppFrontendURL, "https://") {
		return fmt.Errorf("APP_FRONTEND_URL must use https when ENV=production")
	}
	if strings.Contains(c.DatabaseURL, "sslmode=disable") {
		return fmt.Errorf("DATABASE_URL must not use sslmode=disable when ENV=production")
	}
	if strings.HasPrefix(c.StripeSecretKey, "sk_test_") {
		return fmt.Errorf("STRIPE_SECRET_KEY must be a live key when ENV=production")
	}
	if c.AuthJWTSecret == "" || c.AuthJWTSecret == "dev-only-change-me" {
		return fmt.Errorf("AUTH_JWT_SECRET must be set to a strong secret when ENV=production")
	}
	if c.MetricsEnabled && c.MetricsAPIKey == "" {
		return fmt.Errorf("METRICS_API_KEY is required when METRICS_ENABLED=true in production")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
