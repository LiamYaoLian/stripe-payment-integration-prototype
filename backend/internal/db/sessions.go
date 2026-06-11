package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// UserSession is a server-side authenticated session.
type UserSession struct {
	ID         string
	CustomerID string
	Email      string
	ExpiresAt  time.Time
}

// CreateUserSession inserts a new session for a customer.
func (s *Store) CreateUserSession(ctx context.Context, id, tokenHash, customerID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_sessions (id, token_hash, customer_id, expires_at)
		VALUES ($1, $2, $3, $4)`,
		id, tokenHash, customerID, expiresAt,
	)
	return err
}

// GetUserSessionByTokenHash returns an active session for a token hash.
func (s *Store) GetUserSessionByTokenHash(ctx context.Context, tokenHash string) (*UserSession, error) {
	var session UserSession
	err := s.pool.QueryRow(ctx, `
		SELECT us.id, us.customer_id, c.email, us.expires_at
		FROM user_sessions us
		JOIN customers c ON c.id = us.customer_id
		WHERE us.token_hash = $1
			AND us.revoked_at IS NULL
			AND us.expires_at > now()`,
		tokenHash,
	).Scan(&session.ID, &session.CustomerID, &session.Email, &session.ExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// TouchUserSession extends session expiry.
func (s *Store) TouchUserSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET expires_at = $2
		WHERE id = $1 AND revoked_at IS NULL`,
		sessionID, expiresAt,
	)
	return err
}

// RevokeUserSession marks a session as revoked.
func (s *Store) RevokeUserSession(ctx context.Context, sessionID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL`,
		sessionID,
	)
	return err
}

// RevokeAllUserSessions revokes every session for a customer.
func (s *Store) RevokeAllUserSessions(ctx context.Context, customerID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = now()
		WHERE customer_id = $1 AND revoked_at IS NULL`,
		customerID,
	)
	return err
}
