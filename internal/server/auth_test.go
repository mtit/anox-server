package server

import (
	"testing"
	"time"
)

func TestAccessTokenValidation(t *testing.T) {
	now := time.Unix(1700000000, 0)
	token, err := issueAccessToken("secret-a", now)
	if err != nil {
		t.Fatalf("issueAccessToken() error = %v", err)
	}

	if !validateAccessToken(token, "secret-a", now.Add(11*time.Hour)) {
		t.Fatal("expected token to be valid before expiry")
	}
	if validateAccessToken(token, "secret-a", now.Add(13*time.Hour)) {
		t.Fatal("expected token to expire after ttl")
	}
	if validateAccessToken(token, "secret-b", now.Add(time.Hour)) {
		t.Fatal("expected token to be invalid after secret changes")
	}
}
