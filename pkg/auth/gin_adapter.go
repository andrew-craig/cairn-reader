package auth

import (
	"crypto/rsa"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PublicKeyProvider is an interface for types that can provide an RSA public key
// This allows the middleware to work with any JWT manager implementation
type PublicKeyProvider interface {
	GetPublicKey() *rsa.PublicKey
}

// GinMiddleware provides Gin-compatible wrappers around the stdlib auth middleware
type GinMiddleware struct {
	validator *Validator
}

// NewGinMiddleware creates a new Gin middleware adapter
// This wraps the stdlib authentication middleware for use with Gin framework
// Accepts any type that can provide a public key (e.g., JWTManager from services/users/internal/auth)
func NewGinMiddleware(keyProvider PublicKeyProvider) *GinMiddleware {
	return &GinMiddleware{
		validator: NewValidator(keyProvider.GetPublicKey()),
	}
}

// JWTAuth creates a Gin middleware that validates JWT tokens
// This is the primary middleware for protecting endpoints
func (gm *GinMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		tokenString, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			if errors.Is(err, ErrMissingToken) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "missing token",
				})
			} else if errors.Is(err, ErrInvalidAuthHeader) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid authorization header format, expected 'Bearer <token>'",
				})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid authorization header",
				})
			}
			c.Abort()
			return
		}

		// Validate token
		claims, err := gm.validator.ValidateToken(tokenString)
		if err != nil {
			// Return appropriate error based on type
			if errors.Is(err, ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "token has expired",
				})
			} else if errors.Is(err, ErrInvalidSignature) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid token signature",
				})
			} else if errors.Is(err, ErrInvalidIssuer) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid token issuer",
				})
			} else if errors.Is(err, ErrInvalidAudience) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid token audience",
				})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid token",
				})
			}
			c.Abort()
			return
		}

		// Store user ID in Gin context for use by handlers
		c.Set(string(UserIDContextKey), claims.UserID)

		// Continue to next handler
		c.Next()
	}
}

// OptionalAuth creates a middleware that validates JWT tokens if present but doesn't require them
// This is useful for endpoints that behave differently for authenticated vs unauthenticated users
func (gm *GinMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		// Try to extract and validate token
		tokenString, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			// Invalid header format, but we'll allow the request to continue
			c.Next()
			return
		}

		// Validate token
		claims, err := gm.validator.ValidateToken(tokenString)
		if err != nil {
			// Invalid token, but we'll allow the request to continue
			c.Next()
			return
		}

		// Store user ID in context if token is valid
		c.Set(string(UserIDContextKey), claims.UserID)

		c.Next()
	}
}

// GetUserIDFromGinContext extracts the user ID from the Gin context
// This should be called after the JWTAuth middleware has run
func GetUserIDFromGinContext(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get(string(UserIDContextKey))
	if !exists {
		return uuid.Nil, errors.New("user ID not found in context")
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid user ID type in context")
	}

	return id, nil
}

// MustGetUserIDFromGin extracts the user ID from Gin context and panics if not found
// Only use this after JWTAuth middleware where user ID is guaranteed to exist
func MustGetUserIDFromGin(c *gin.Context) uuid.UUID {
	userID, err := GetUserIDFromGinContext(c)
	if err != nil {
		panic("user ID not found in context - ensure JWTAuth middleware is applied")
	}
	return userID
}

// IsAuthenticatedInGin checks if the current request is authenticated
func IsAuthenticatedInGin(c *gin.Context) bool {
	_, exists := c.Get(string(UserIDContextKey))
	return exists
}

// RequireAuth is an alias for JWTAuth for backward compatibility and clarity
func (gm *GinMiddleware) RequireAuth() gin.HandlerFunc {
	return gm.JWTAuth()
}
