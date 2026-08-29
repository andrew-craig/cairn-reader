// Package middleware provides HTTP middleware for the email ingest service.
package middleware

import (
	"crypto/rsa"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
)

// JWTAuth provides JWT authentication middleware for user-facing endpoints.
// Validates JWT tokens using a public key from Vault.
type JWTAuth struct {
	*auth.Middleware
}

// NewJWTAuth creates a new JWT authentication middleware using the given RSA public key.
func NewJWTAuth(publicKey *rsa.PublicKey) *JWTAuth {
	validator := auth.NewValidator(publicKey)
	return &JWTAuth{
		Middleware: auth.NewMiddleware(validator),
	}
}

// NewJWTAuthFromValidator creates a new JWT authentication middleware from an existing validator.
func NewJWTAuthFromValidator(validator *auth.Validator) *JWTAuth {
	return &JWTAuth{
		Middleware: auth.NewMiddleware(validator),
	}
}
