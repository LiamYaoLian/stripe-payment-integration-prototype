package config

import "github.com/joho/godotenv"

// LoadSeed reads only the database URL for the seed command.
// It does not require Stripe credentials.
func LoadSeed() (string, error) {
	_ = godotenv.Load()
	return getEnv("DATABASE_URL", "postgresql://stripe:stripe@localhost:5434/stripe_payment?sslmode=disable"), nil
}
