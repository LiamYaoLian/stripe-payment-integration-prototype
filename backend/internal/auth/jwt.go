package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleUser = "user"
)

// UserClaims is the JWT payload for authenticated user sessions.
type UserClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// IssueUserToken signs a user session JWT.
func IssueUserToken(secret, customerID, email string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	expiresAt = time.Now().UTC().Add(ttl)
	claims := UserClaims{
		Email: email,
		Role:  RoleUser,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   customerID,
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

// VerifyUserToken validates a user JWT and returns its claims.
func VerifyUserToken(secret, token string) (*UserClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*UserClaims)
	if !ok || !parsed.Valid || claims.Role != RoleUser || claims.Email == "" || claims.Subject == "" {
		return nil, fmt.Errorf("invalid user token")
	}
	return claims, nil
}
