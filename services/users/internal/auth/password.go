// Package auth provides authentication and authorization utilities for the user service.
// This file handles password hashing using bcrypt and password strength validation.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher handles password hashing operations using bcrypt.
// The cost parameter controls the computational expense of hashing,
// with higher values being more secure but slower. A minimum cost of 12
// is recommended for production use.
type PasswordHasher struct {
	cost int // bcrypt cost factor (4-31, recommended 12+)
}

// NewPasswordHasher creates a new password hasher with the specified bcrypt cost.
// The cost should be at least 12 for production environments to ensure adequate
// protection against brute-force attacks.
func NewPasswordHasher(cost int) *PasswordHasher {
	return &PasswordHasher{cost: cost}
}

// HashPassword hashes a password using bcrypt
func (p *PasswordHasher) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// ComparePassword compares a password with its hash
func (p *PasswordHasher) ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePasswordStrength validates that a password meets minimum security requirements.
// When requireComplexity is true, the password must contain:
//   - At least one uppercase letter (A-Z)
//   - At least one lowercase letter (a-z)
//   - At least one digit (0-9)
//   - At least one special character (!@#$%^&*()_+-=[]{}|;':",.<>?/`~)
func ValidatePasswordStrength(password string, minLength int, requireComplexity bool) error {
	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters long", minLength)
	}

	if requireComplexity {
		hasUpper := false
		hasLower := false
		hasDigit := false
		hasSpecial := false

		for _, char := range password {
			switch {
			case char >= 'A' && char <= 'Z':
				hasUpper = true
			case char >= 'a' && char <= 'z':
				hasLower = true
			case char >= '0' && char <= '9':
				hasDigit = true
			// Special characters are defined by ASCII ranges:
			// 33-47:  ! " # $ % & ' ( ) * + , - . /
			// 58-64:  : ; < = > ? @
			// 91-96:  [ \ ] ^ _ `
			// 123-126: { | } ~
			case char >= 33 && char <= 47 || char >= 58 && char <= 64 || char >= 91 && char <= 96 || char >= 123 && char <= 126:
				hasSpecial = true
			}
		}

		if !hasUpper || !hasLower || !hasDigit {
			return fmt.Errorf("password must contain uppercase, lowercase, and numeric characters")
		}

		if !hasSpecial {
			return fmt.Errorf("password must contain at least one special character")
		}
	}

	return nil
}
