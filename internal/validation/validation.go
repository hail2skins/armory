package validation

import (
	"errors"
	"regexp"
)

var (
	// ErrInvalidEmail is returned when an email format is invalid
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrPasswordTooShort is returned when a password is too short
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")

	// ErrPasswordNoUppercase is returned when a password has no uppercase letters
	ErrPasswordNoUppercase = errors.New("password must contain at least one uppercase letter")

	// ErrPasswordNoSpecialChar is returned when a password has no special characters
	ErrPasswordNoSpecialChar = errors.New("password must contain at least one special character")
)

// EmailValidator validates email addresses
func ValidateEmail(email string) error {
	// Use a regular expression to validate email format
	// This is a simplified regex for email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword checks if a password meets the requirements:
// - At least 8 characters
// - At least one uppercase letter
// - At least one special character
func ValidatePassword(password string) error {
	// Check length
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	// Check for uppercase
	uppercaseRegex := regexp.MustCompile(`[A-Z]`)
	if !uppercaseRegex.MatchString(password) {
		return ErrPasswordNoUppercase
	}

	// Check for special character
	specialCharRegex := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
	if !specialCharRegex.MatchString(password) {
		return ErrPasswordNoSpecialChar
	}

	return nil
}
