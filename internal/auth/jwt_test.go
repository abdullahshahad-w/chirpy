package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned an unexpected error: %v", err)
	}

	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse generated token: %v", err)
	}

	if !token.Valid {
		t.Fatal("Expected token to be valid")
	}

	if claims.Subject != userID.String() {
		t.Errorf("Expected subject %q, got %q", userID.String(), claims.Subject)
	}

	if claims.Issuer != "chirpy-access" {
		t.Errorf("Expected issuer %q, got %q", "chirpy-access", claims.Issuer)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("Expected expiration claim to be set")
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("Expected expiration time to be in the future")
	}

	if claims.IssuedAt == nil {
		t.Fatal("Expected issued-at claim to be set")
	}
}