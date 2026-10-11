package auth

import (
	"errors"
	"testing"
)

func TestValidation_Email(t *testing.T) {
	validEmails := []string{
		"test@example.com",
		"user.name+tag@domain.co.uk",
		"first.last@company.gh",
	}

	for _, email := range validEmails {
		if err := ValidateEmail(email); err != nil {
			t.Errorf("expected email %q to be valid, got: %v", email, err)
		}
	}

	invalidEmails := []string{
		"",
		"not-an-email",
		"@missing-user.com",
		"missing-domain@",
		"missing-dot@domain",
	}

	for _, email := range invalidEmails {
		if err := ValidateEmail(email); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("expected ErrInvalidEmail for %q, got: %v", email, err)
		}
	}
}

func TestValidation_Password(t *testing.T) {
	// Valid password: length >= 8, uppercase, lowercase, digit
	if err := ValidatePassword("StrongP@ss1"); err != nil {
		t.Errorf("expected valid password, got: %v", err)
	}

	// Too short (< 8 chars)
	if err := ValidatePassword("Ab1!"); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got: %v", err)
	}

	// Missing uppercase
	if err := ValidatePassword("lowercase123"); !errors.Is(err, ErrPasswordNeedsUpper) {
		t.Errorf("expected ErrPasswordNeedsUpper, got: %v", err)
	}

	// Missing lowercase
	if err := ValidatePassword("UPPERCASE123"); !errors.Is(err, ErrPasswordNeedsLower) {
		t.Errorf("expected ErrPasswordNeedsLower, got: %v", err)
	}

	// Missing digit
	if err := ValidatePassword("NoDigitsPassword"); !errors.Is(err, ErrPasswordNeedsDigit) {
		t.Errorf("expected ErrPasswordNeedsDigit, got: %v", err)
	}
}
