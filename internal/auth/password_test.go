package auth

import (
	"testing"
)

func TestPassword_HashAndVerify(t *testing.T) {
	password := "SecureP@ssw0rd123"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("expected different salts and hashes for identical passwords")
	}

	// Verify correct password
	match, err := VerifyPassword(password, hash1)
	if err != nil {
		t.Fatalf("unexpected error verifying password: %v", err)
	}
	if !match {
		t.Errorf("expected password to match hash")
	}

	// Verify wrong password
	match, err = VerifyPassword("WrongPassword!", hash1)
	if err != nil {
		t.Fatalf("unexpected error verifying wrong password: %v", err)
	}
	if match {
		t.Errorf("expected wrong password not to match")
	}

	// Verify corrupted hash
	_, err = VerifyPassword(password, "invalid_hash_string")
	if err == nil {
		t.Errorf("expected error for malformed hash, got nil")
	}
}
