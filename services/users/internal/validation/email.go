// Package validation provides input validation utilities.
package validation

import (
	"fmt"
	"net/mail"
	"strings"
)

// ValidateEmail checks that the given string is a valid RFC 5322 email address.
// It returns an error if the email is empty or not well-formed.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}

	// Reject display-name form like "Name <user@example.com>"
	if addr.Name != "" {
		return fmt.Errorf("invalid email format")
	}

	// Reject bare local-only addresses (no dot in domain part is technically
	// valid per RFC but we want a real domain with a TLD).
	// mail.ParseAddress accepts "user@localhost" which we want to reject in
	// practice for a real application.
	at := strings.LastIndex(addr.Address, "@")
	domain := addr.Address[at+1:]
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("invalid email format")
	}

	return nil
}
