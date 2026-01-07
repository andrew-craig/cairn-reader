package middleware

import (
	"net/http"

	"github.com/andrew-craig/cairn/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireSameUser ensures that the authenticated user can only access their own resources
// This middleware expects a URL parameter named "id" representing the user ID being accessed
// and checks that it matches the authenticated user's ID from the JWT token
func RequireSameUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the JWT token (set by JWTAuth middleware)
		authenticatedUserID, err := auth.GetUserIDFromGinContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Get the user ID from the URL parameter
		requestedUserIDStr := c.Param("id")
		if requestedUserIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "user ID parameter is required",
			})
			c.Abort()
			return
		}

		// Parse the requested user ID
		requestedUserID, err := uuid.Parse(requestedUserIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid user ID format",
			})
			c.Abort()
			return
		}

		// Check if the authenticated user matches the requested user
		if authenticatedUserID != requestedUserID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "you can only access your own resources",
			})
			c.Abort()
			return
		}

		// User is authorized to access this resource
		c.Next()
	}
}

// RequireSameUserWithCustomParam is similar to RequireSameUser but allows specifying
// a custom parameter name instead of "id"
func RequireSameUserWithCustomParam(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the JWT token (set by JWTAuth middleware)
		authenticatedUserID, err := auth.GetUserIDFromGinContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Get the user ID from the URL parameter
		requestedUserIDStr := c.Param(paramName)
		if requestedUserIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "user ID parameter is required",
			})
			c.Abort()
			return
		}

		// Parse the requested user ID
		requestedUserID, err := uuid.Parse(requestedUserIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid user ID format",
			})
			c.Abort()
			return
		}

		// Check if the authenticated user matches the requested user
		if authenticatedUserID != requestedUserID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "you can only access your own resources",
			})
			c.Abort()
			return
		}

		// User is authorized to access this resource
		c.Next()
	}
}

// RequireOwnership is a flexible authorization middleware that takes a function
// to determine if the authenticated user owns the resource
// The ownerCheckFunc should return the owner's user ID and any error
type OwnerCheckFunc func(c *gin.Context) (uuid.UUID, error)

func RequireOwnership(ownerCheckFunc OwnerCheckFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the JWT token (set by JWTAuth middleware)
		authenticatedUserID, err := auth.GetUserIDFromGinContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Get the resource owner ID using the provided function
		resourceOwnerID, err := ownerCheckFunc(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		// Check if the authenticated user owns the resource
		if authenticatedUserID != resourceOwnerID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "you can only access your own resources",
			})
			c.Abort()
			return
		}

		// User is authorized to access this resource
		c.Next()
	}
}

// CheckUserIDParam validates that a user ID parameter exists and is valid
// This is a helper middleware that can be used before authorization checks
func CheckUserIDParam(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.Param(paramName)
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "user ID parameter is required",
			})
			c.Abort()
			return
		}

		_, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid user ID format",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
