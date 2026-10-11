package auth

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

var (
	ErrInvalidEmail       = errors.New("invalid email address format")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters long")
	ErrPasswordNeedsUpper = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNeedsLower = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNeedsDigit = errors.New("password must contain at least one numeric digit")
)

// ValidateEmail checks whether the provided email string has a valid RFC 5322 format.
func ValidateEmail(email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return ErrInvalidEmail
	}

	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return ErrInvalidEmail
	}

	parts := strings.Split(trimmed, "@")
	if len(parts) != 2 || !strings.Contains(parts[1], ".") {
		return ErrInvalidEmail
	}

	return nil
}

// ValidatePassword enforces security policy: min 8 chars, 1 uppercase, 1 lowercase, 1 digit.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	if !hasUpper {
		return ErrPasswordNeedsUpper
	}
	if !hasLower {
		return ErrPasswordNeedsLower
	}
	if !hasDigit {
		return ErrPasswordNeedsDigit
	}

	return nil
}
