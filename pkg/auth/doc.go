// Package auth is a lightweight shared library for JWT validation.
// This package is distributed to other Cairn services (content, recommendations)
// to enable stateless JWT validation using the RS256 public key.
// It provides middleware for protecting endpoints and utilities for
// extracting user context from validated tokens.
package auth
