// Package handlers provides HTTP request handlers for the user service API.
// It sets up routes, middleware, and connects handlers to their respective services.
package handlers

import (
	"log/slog"
	"time"

	"github.com/andrew-craig/cairn/pkg/auth"
	"github.com/andrew-craig/cairn/pkg/logging"
	localAuth "github.com/andrew-craig/cairn/services/users/internal/auth"
	"github.com/andrew-craig/cairn/services/users/internal/database"
	"github.com/andrew-craig/cairn/services/users/internal/middleware"
	"github.com/andrew-craig/cairn/services/users/internal/services"
	"github.com/gin-gonic/gin"
)

// RouterConfig holds all dependencies needed to set up the HTTP router.
// These dependencies are injected from the main application during startup.
type RouterConfig struct {
	DB                  *database.DB             // Database connection for health checks
	VaultClient         *localAuth.VaultClient   // Vault client for health checks
	AuthService         services.AuthService     // Service handling authentication operations
	UserService         services.UserService     // Service handling user management operations
	JWTManager          *localAuth.JWTManager    // JWT manager for token validation middleware
	AuthRateLimit       int                      // Requests per window for auth endpoints (default: 10)
	AuthRateLimitWindow time.Duration            // Time window for auth rate limiting (default: 1 minute)
	Logger              *slog.Logger             // Structured logger for request logging
}

// Router sets up the HTTP routes and returns a configured gin.Engine.
// It applies the following middleware chain to all routes:
//   - Recovery: Recovers from panics and returns 500 errors
//   - RequestLogger: Structured logging with slog for all requests
//   - CORS: Handles Cross-Origin Resource Sharing
//   - RequireHTTPS: Enforces HTTPS in production
//   - SecureHeadersRelaxed: Adds security headers (CSP, X-Frame-Options, etc.)
func Router(config RouterConfig) *gin.Engine {
	router := gin.New() // Use gin.New() instead of gin.Default() to avoid default logger

	// Apply global middleware stack for security and observability
	router.Use(middleware.Recovery())
	router.Use(logging.RequestLogger(config.Logger)) // Use structured logging middleware
	router.Use(middleware.CORS())
	router.Use(middleware.RequireHTTPS())
	router.Use(middleware.SecureHeadersRelaxed())

	// Initialize handlers
	healthHandler := NewHealthHandler(config.DB, config.VaultClient)
	authHandler := NewAuthHandler(config.AuthService)
	userHandler := NewUserHandler(config.UserService)

	// Create Gin middleware adapter from pkg/auth
	// This provides Gin-compatible wrappers around the stdlib auth middleware
	authMiddleware := auth.NewGinMiddleware(config.JWTManager)

	// Health endpoints (Kubernetes-compatible)
	router.GET("/health/live", healthHandler.LivenessCheck)
	router.GET("/health/ready", healthHandler.ReadinessCheck)

	// Set default rate limit values if not provided
	authRateLimit := config.AuthRateLimit
	if authRateLimit == 0 {
		authRateLimit = 10 // Default: 10 requests per window
	}
	authRateLimitWindow := config.AuthRateLimitWindow
	if authRateLimitWindow == 0 {
		authRateLimitWindow = 1 * time.Minute // Default: 1 minute window
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication endpoints - public routes with rate limiting to prevent brute force attacks
		// Rate limiting is applied per IP address to mitigate credential stuffing and enumeration attacks
		authGroup := v1.Group("/auth")
		authGroup.Use(middleware.RateLimitAuth(authRateLimit, authRateLimitWindow))
		{
			authGroup.POST("/register", authHandler.Register)           // Create account with email/password
			authGroup.POST("/register/mobile", authHandler.RegisterMobile) // Create mobile-only account with device ID
			authGroup.POST("/login", authHandler.Login)                 // Authenticate with email/password
			authGroup.POST("/login/mobile", authHandler.LoginMobile)    // Authenticate with device ID
			authGroup.POST("/refresh", authHandler.Refresh)             // Exchange refresh token for new access token
			authGroup.POST("/logout", authHandler.Logout)               // Revoke a specific refresh token
			// logout-all requires authentication since it needs to know which user's tokens to revoke
			authGroup.POST("/logout-all", authMiddleware.JWTAuth(), authHandler.LogoutAll)
		}

		// User management endpoints - all routes require JWT authentication
		// Authorization (ensuring users can only access their own data) is handled in the service layer
		// Note: Using :user_id parameter for consistency with API standards
		users := v1.Group("/user")
		users.Use(authMiddleware.JWTAuth())
		{
			users.GET("/:user_id", userHandler.GetUser)           // Get user profile
			users.PATCH("/:user_id", userHandler.UpdateUser)      // Update user email
			users.POST("/:user_id/upgrade", userHandler.UpgradeAccount) // Add email/password to mobile-only account
			users.DELETE("/:user_id", userHandler.DeleteUser)     // Delete user account and all associated data
		}
	}

	return router
}
