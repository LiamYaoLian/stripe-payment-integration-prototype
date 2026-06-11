package auth

import (
	"testing"
	"time"
)

func TestUserJWTIssueAndVerify(t *testing.T) {
	token, expiresAt, err := IssueUserToken("test-secret", "cust_1", "buyer@example.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expiresAt in the past")
	}
	claims, err := VerifyUserToken("test-secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "cust_1" || claims.Email != "buyer@example.com" || claims.Role != RoleUser {
		t.Fatalf("claims %+v", claims)
	}
}
