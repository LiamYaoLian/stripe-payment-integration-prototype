package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleGuest = "guest"
)

// GuestClaims is the JWT payload for email-scoped guest sessions.
type GuestClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// IssueGuestToken signs a short-lived guest session JWT.
func IssueGuestToken(secret, email string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	expiresAt = time.Now().UTC().Add(ttl)
	claims := GuestClaims{
		Email: email,
		Role:  RoleGuest,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// VerifyGuestToken validates a guest JWT and returns its claims.
func VerifyGuestToken(secret, token string) (*GuestClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &GuestClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*GuestClaims)
	if !ok || !parsed.Valid || claims.Role != RoleGuest || claims.Email == "" {
		return nil, fmt.Errorf("invalid guest token")
	}
	return claims, nil
}
