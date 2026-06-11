package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ListActiveProducts returns all active catalog products.
func (s *Store) ListActiveProducts(ctx context.Context) ([]Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, unit_amount_cents, currency, active
		FROM products WHERE active = true ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.UnitAmountCents, &product.Currency, &product.Active); err != nil {
			return nil, err
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
		return nil, err
	}
	return &product, nil
}

// UpsertProduct inserts a product if it does not already exist.
func (s *Store) UpsertProduct(ctx context.Context, product Product) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO products (id, name, description, unit_amount_cents, currency, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`,
		product.ID, product.Name, product.Description, product.UnitAmountCents, product.Currency, product.Active)
	return err
}
