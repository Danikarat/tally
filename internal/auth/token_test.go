package auth

import (
	"testing"
	"time"
)

func TestToken_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-at-least-32-chars-long"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "user@example.com"
	duration := 1 * time.Hour

	tokenString, expiresAt, err := GenerateToken(userID, email, secret, duration)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if tokenString == "" {
		t.Fatal("expected non-empty token string")
	}

	if expiresAt.Before(time.Now()) {
		t.Fatal("expected expiration time to be in the future")
	}

	// Validate valid token
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		t.Fatalf("failed to validate valid token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %q, got %q", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected email %q, got %q", email, claims.Email)
	}

	// Validate with wrong secret
	_, err = ValidateToken(tokenString, "wrong-secret-key-32-chars-long-here")
	if err == nil {
		t.Error("expected error validating token with wrong secret, got nil")
	}

	// Validate expired token
	expiredToken, _, err := GenerateToken(userID, email, secret, -1*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	_, err = ValidateToken(expiredToken, secret)
	if err == nil {
		t.Error("expected error validating expired token, got nil")
	}
}
