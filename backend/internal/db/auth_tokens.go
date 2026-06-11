package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreatePasswordResetToken stores a password reset token hash.
func (s *Store) CreatePasswordResetToken(ctx context.Context, id, tokenHash, customerID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (id, token_hash, customer_id, expires_at)
		VALUES ($1, $2, $3, $4)`,
		id, tokenHash, customerID, expiresAt,
	)
	return err
}

// ConsumePasswordResetToken marks a valid reset token as used and returns the customer ID.
func (s *Store) ConsumePasswordResetToken(ctx context.Context, tokenHash string) (string, error) {
	var customerID string
	err := s.pool.QueryRow(ctx, `
		UPDATE password_reset_tokens
		SET used_at = now()
		WHERE token_hash = $1
			AND used_at IS NULL
			AND expires_at > now()
		RETURNING customer_id`,
		tokenHash,
	).Scan(&customerID)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return customerID, nil
}

// CreateEmailVerificationToken stores an email verification token hash.
func (s *Store) CreateEmailVerificationToken(ctx context.Context, id, tokenHash, customerID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (id, token_hash, customer_id, expires_at)
		VALUES ($1, $2, $3, $4)`,
		id, tokenHash, customerID, expiresAt,
	)
	return err
}

// ConsumeEmailVerificationToken marks a valid verification token as used and returns the customer ID.
func (s *Store) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (string, error) {
	var customerID string
	err := s.pool.QueryRow(ctx, `
		UPDATE email_verification_tokens
		SET used_at = now()
		WHERE token_hash = $1
			AND used_at IS NULL
			AND expires_at > now()
		RETURNING customer_id`,
		tokenHash,
	).Scan(&customerID)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return customerID, nil
}

// MarkCustomerEmailVerified sets email_verified_at for a customer.
func (s *Store) MarkCustomerEmailVerified(ctx context.Context, customerID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE customers SET email_verified_at = now()
		WHERE id = $1 AND email_verified_at IS NULL`,
		customerID,
	)
	return err
}

// UpdateCustomerPassword replaces the password hash for a customer.
func (s *Store) UpdateCustomerPassword(ctx context.Context, customerID, passwordHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE customers SET password_hash = $2
		WHERE id = $1`,
		customerID, passwordHash,
	)
	return err
}
