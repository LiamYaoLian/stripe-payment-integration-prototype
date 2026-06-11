package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ListActiveProducts returns all active catalog products.
func (s *Store) ListActiveProducts(ctx context.Context) ([]Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, unit_amount_cents, currency, active
		FROM products WHERE active = true ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.UnitAmountCents, &product.Currency, &product.Active); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

// GetProduct returns an active product by ID.
func (s *Store) GetProduct(ctx context.Context, id string) (*Product, error) {
	var product Product
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, description, unit_amount_cents, currency, active
		FROM products WHERE id = $1 AND active = true`, id).
		Scan(&product.ID, &product.Name, &product.Description, &product.UnitAmountCents, &product.Currency, &product.Active)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get product %q: %w", id, err)
	}
	return &product, nil
}

// GetProductsByIDs returns active products keyed by ID.
func (s *Store) GetProductsByIDs(ctx context.Context, ids []string) (map[string]Product, error) {
	if len(ids) == 0 {
		return map[string]Product{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, unit_amount_cents, currency, active
		FROM products WHERE id = ANY($1) AND active = true`, ids)
	if err != nil {
		return nil, fmt.Errorf("get products by ids: %w", err)
	}
	defer rows.Close()

	products := make(map[string]Product, len(ids))
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.UnitAmountCents, &product.Currency, &product.Active); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products[product.ID] = product
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return products, nil
}

// UpsertProduct inserts a product if it does not already exist.
func (s *Store) UpsertProduct(ctx context.Context, product Product) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO products (id, name, description, unit_amount_cents, currency, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`,
		product.ID, product.Name, product.Description, product.UnitAmountCents, product.Currency, product.Active)
	if err != nil {
		return fmt.Errorf("upsert product %q: %w", product.ID, err)
	}
	return nil
}
