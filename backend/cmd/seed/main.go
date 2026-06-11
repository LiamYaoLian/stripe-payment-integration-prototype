package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/config"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

func main() {
	databaseURL, err := config.LoadSeed()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := db.New(ctx, databaseURL)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	desc1 := "One-time purchase for personal use"
	desc2 := "Team license for up to 10 members"
	desc3 := "Support the project"

	products := []db.Product{
		{ID: "d4j8k2m9q1p7n3s6", Name: "Pro License", Description: &desc1, UnitAmountCents: 4900, Currency: "usd", Active: true},
		{ID: "d4j8k2m9q1p7n3s7", Name: "Team License", Description: &desc2, UnitAmountCents: 19900, Currency: "usd", Active: true},
		{ID: "d4j8k2m9q1p7n3s8", Name: "Donation", Description: &desc3, UnitAmountCents: 500, Currency: "usd", Active: true},
	}

	for _, product := range products {
		if err := store.UpsertProduct(ctx, product); err != nil {
			slog.Error("seed product failed", "product", product.Name, "error", err)
			os.Exit(1)
		}
		slog.Info("seeded product", "name", product.Name, "id", product.ID)
	}
}
