package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Customer is a registered user account.
type Customer struct {
	ID               string
	Email            string
	PasswordHash     string
	StripeCustomerID *string
	EmailVerifiedAt  *time.Time
	CreatedAt        time.Time
}

// CreateCustomer inserts a new customer account.
func (s *Store) CreateCustomer(ctx context.Context, id, email, passwordHash string) (*Customer, error) {
	var customer Customer
	err := s.pool.QueryRow(ctx, `
		INSERT INTO customers (id, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, stripe_customer_id, email_verified_at, created_at`,
		id, email, passwordHash,
	).Scan(&customer.ID, &customer.Email, &customer.PasswordHash, &customer.StripeCustomerID, &customer.EmailVerifiedAt, &customer.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetCustomerByEmail returns a customer by email (case-insensitive).
func (s *Store) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	var customer Customer
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, stripe_customer_id, email_verified_at, created_at
		FROM customers WHERE lower(email) = lower($1)`, email,
	).Scan(&customer.ID, &customer.Email, &customer.PasswordHash, &customer.StripeCustomerID, &customer.EmailVerifiedAt, &customer.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetCustomerByID returns a customer by primary key.
func (s *Store) GetCustomerByID(ctx context.Context, id string) (*Customer, error) {
	var customer Customer
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, stripe_customer_id, email_verified_at, created_at
		FROM customers WHERE id = $1`, id,
	).Scan(&customer.ID, &customer.Email, &customer.PasswordHash, &customer.StripeCustomerID, &customer.EmailVerifiedAt, &customer.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &customer, nil
}
