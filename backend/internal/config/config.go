package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	CORSOrigin        string
	AppFrontendURL    string
	StripeSecretKey   string
	StripeWebhookSecret string
	DatabaseURL       string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		CORSOrigin:        getEnv("CORS_ORIGIN", "http://localhost:5173"),
		AppFrontendURL:    getEnv("APP_FRONTEND_URL", "http://localhost:5173"),
		StripeSecretKey:   os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgresql://stripe:stripe@localhost:5434/stripe_payment?sslmode=disable"),
	}

	if cfg.StripeSecretKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
