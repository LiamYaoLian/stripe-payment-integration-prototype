package main

import (
	"context"
	"log"
	"os"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/config"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	store, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
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

	for _, p := range products {
		if err := store.UpsertProduct(ctx, p); err != nil {
			log.Fatal(err)
		}
		log.Printf("seeded product: %s (%s)", p.Name, p.ID)
	}
	os.Exit(0)
}
