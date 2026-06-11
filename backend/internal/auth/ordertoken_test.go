package auth

import "testing"

func TestVerifyOrderAccessToken(t *testing.T) {
	token, hash, err := GenerateOrderAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyOrderAccessToken(token, hash) {
		t.Fatal("expected token to verify")
	}
	if VerifyOrderAccessToken("wrong", hash) {
		t.Fatal("expected wrong token to fail")
	}
}
