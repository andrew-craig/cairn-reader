package middleware

import (
	"errors"
	"net/http"

	"github.com/andrew-craig/cairn-core/user-service/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Context keys for storing values in gin.Context
const (
	UserIDKey = "user_id"
)

// JWTAuth creates a middleware that validates JWT tokens
func JWTAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
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
		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			if errors.Is(err, auth.ErrMissingToken) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "missing token",
				})
			} else if errors.Is(err, auth.ErrInvalidAuthHeader) {
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
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			// Return appropriate error based on type
			if errors.Is(err, auth.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "token has expired",
				})
			} else if errors.Is(err, auth.ErrInvalidSignature) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid token signature",
				})
			} else if errors.Is(err, auth.ErrInvalidIssuer) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid token issuer",
				})
			} else if errors.Is(err, auth.ErrInvalidAudience) {
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

		// Store user ID in context for use by handlers
		c.Set(UserIDKey, claims.UserID)

		// Continue to next handler
		c.Next()
	}
}

// GetUserIDFromContext extracts the user ID from the gin context
// This should be called after the JWTAuth middleware has run
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return uuid.Nil, errors.New("user ID not found in context")
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid user ID type in context")
	}

	return id, nil
}

// RequireAuth is an alias for JWTAuth for backward compatibility and clarity
func RequireAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return JWTAuth(jwtManager)
}

// OptionalAuth creates a middleware that validates JWT tokens if present but doesn't require them
// This is useful for endpoints that behave differently for authenticated vs unauthenticated users
func OptionalAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		// Try to extract and validate token
		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			// Invalid header format, but we'll allow the request to continue
			c.Next()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			// Invalid token, but we'll allow the request to continue
			c.Next()
			return
		}

		// Store user ID in context if token is valid
		c.Set(UserIDKey, claims.UserID)

		c.Next()
	}
}

// IsAuthenticated checks if the current request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get(UserIDKey)
	return exists
}

// MustGetUserID extracts the user ID from context and panics if not found
// Only use this after JWTAuth middleware where user ID is guaranteed to exist
func MustGetUserID(c *gin.Context) uuid.UUID {
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		panic("user ID not found in context - ensure JWTAuth middleware is applied")
	}
	return userID
}
