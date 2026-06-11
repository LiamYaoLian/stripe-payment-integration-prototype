package service

import (
	"context"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

type productStore interface {
	ListActiveProducts(ctx context.Context) ([]db.Product, error)
}

// ProductService handles catalog read operations.
type ProductService struct {
	store productStore
}

// NewProductService returns a ProductService backed by the given store.
func NewProductService(store productStore) *ProductService {
	return &ProductService{store: store}
}

// ListProducts returns all active products.
func (s *ProductService) ListProducts(ctx context.Context) ([]db.Product, error) {
	return s.store.ListActiveProducts(ctx)
}
