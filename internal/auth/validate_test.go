package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	tokenString, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned an unexpected error: %v", err)
	}

	gotUserID, err := ValidateJWT(tokenString, secret)
	if err != nil {
		t.Fatalf("ValidateJWT returned an unexpected error: %v", err)
	}

	if gotUserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, gotUserID)
	}
}

func TestValidateJWTWrongSecret(t *testing.T) {
	userID := uuid.New()

	tokenString, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned an unexpected error: %v", err)
	}

	_, err = ValidateJWT(tokenString, "wrong-secret")
	if err == nil {
		t.Fatal("Expected an error when using the wrong secret")
	}
}

func TestValidateJWTMalformedToken(t *testing.T) {
	_, err := ValidateJWT("this-is-not-a-jwt", "test-secret")

	if err == nil {
		t.Fatal("Expected an error for malformed token")
	}
}

func TestValidateJWTExpiredToken(t *testing.T) {
	userID := uuid.New()

	tokenString, err := MakeJWT(userID, "test-secret", -time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned an unexpected error: %v", err)
	}

	_, err = ValidateJWT(tokenString, "test-secret")

	if err == nil {
		t.Fatal("Expected an error for expired token")
	}
}

func TestValidateJWTInvalidSubject(t *testing.T) {
	secret := "test-secret"

	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		Subject:   "not-a-valid-uuid",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign test token: %v", err)
	}

	_, err = ValidateJWT(tokenString, secret)

	if err == nil {
		t.Fatal("Expected an error for invalid UUID subject")
	}
}