package auth

import (
	"testing"
	"time"
)

func TestGuestJWTIssueAndVerify(t *testing.T) {
	token, expiresAt, err := IssueGuestToken("test-secret", "buyer@example.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expiresAt in the past")
	}
	claims, err := VerifyGuestToken("test-secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Email != "buyer@example.com" || claims.Role != RoleGuest {
		t.Fatalf("claims %+v", claims)
	}
}
