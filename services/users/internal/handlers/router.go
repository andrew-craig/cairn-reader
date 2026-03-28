// Package handlers provides HTTP request handlers for the user service API.
// It sets up routes, middleware, and connects handlers to their respective services.
package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	localAuth "github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
	"github.com/go-chi/chi/v5"
)

// RouterConfig holds all dependencies needed to set up the HTTP router.
// These dependencies are injected from the main application during startup.
type RouterConfig struct {
	DB                  *database.DB           // Database connection for health checks
	VaultClient         *localAuth.VaultClient // Vault client for health checks
	AuthService         services.AuthService   // Service handling authentication operations
	UserService         services.UserService   // Service handling user management operations
	JWTManager          *localAuth.JWTManager  // JWT manager for token validation middleware
	AuthRateLimit       int                    // Requests per window for auth endpoints (default: 10)
	AuthRateLimitWindow time.Duration          // Time window for auth rate limiting (default: 1 minute)
	Logger              *slog.Logger           // Structured logger for request logging
}

// Router sets up the HTTP routes and returns a configured http.Handler.
// Health endpoints are registered without security middleware to allow internal HTTP checks.
// API endpoints have the following middleware chain:
//   - Recovery: Recovers from panics and returns 500 errors
//   - RequestLogger: Structured logging with slog for all requests
//   - CORS: Handles Cross-Origin Resource Sharing
//   - RequireHTTPS: Enforces HTTPS in production
//   - SecureHeadersRelaxed: Adds security headers (CSP, X-Frame-Options, etc.)
func Router(config RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Apply only recovery and logging globally (needed for all routes including health checks)
	r.Use(sharedmw.Recovery)
	r.Use(logging.ChiRequestLogger(config.Logger))

	// Initialize handlers
	healthHandler := NewHealthHandler(config.DB, config.VaultClient)
	authHandler := NewAuthHandler(config.AuthService)
	userHandler := NewUserHandler(config.UserService)

	// Create stdlib middleware from pkg/auth
	authMiddleware := auth.NewMiddleware(auth.NewValidator(config.JWTManager.GetPublicKey()))

	// Health endpoints (Kubernetes-compatible)
	// No security middleware applied to allow HTTP health checks within Docker
	// Support both GET and HEAD methods for health checks (HEAD used by Docker/wget --spider)
	r.Get("/health/live", healthHandler.LivenessCheck)
	r.Head("/health/live", healthHandler.LivenessCheck)
	r.Get("/health/ready", healthHandler.ReadinessCheck)
	r.Head("/health/ready", healthHandler.ReadinessCheck)

	// Set default rate limit values if not provided
	authRateLimit := config.AuthRateLimit
	if authRateLimit == 0 {
		authRateLimit = 10 // Default: 10 requests per window
	}
	authRateLimitWindow := config.AuthRateLimitWindow
	if authRateLimitWindow == 0 {
		authRateLimitWindow = 1 * time.Minute // Default: 1 minute window
	}

	// API v1 routes with security middleware
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(sharedmw.CORS(sharedmw.DefaultCORSConfig()))
		r.Use(sharedmw.RequireHTTPS)
		r.Use(sharedmw.SecureHeadersRelaxed)

		// Authentication endpoints - public routes with rate limiting to prevent brute force attacks
		// Rate limiting is applied per IP address to mitigate credential stuffing and enumeration attacks
		r.Route("/auth", func(r chi.Router) {
			r.Use(sharedmw.RateLimit(authRateLimit, authRateLimitWindow))

			r.Post("/register", authHandler.Register)              // Create account with email/password
			r.Post("/register/mobile", authHandler.RegisterMobile) // Create mobile-only account with device ID
			r.Post("/login", authHandler.Login)                    // Authenticate with email/password
			r.Post("/login/mobile", authHandler.LoginMobile)       // Authenticate with device ID
			r.Post("/refresh", authHandler.Refresh)                // Exchange refresh token for new access token
			r.Post("/logout", authHandler.Logout)                  // Revoke a specific refresh token
			// logout-all requires authentication since it needs to know which user's tokens to revoke
			r.With(authMiddleware.RequireAuth).Post("/logout-all", authHandler.LogoutAll)
		})

		// User management endpoints - all routes require JWT authentication
		// Authorization (ensuring users can only access their own data) is handled in the service layer
		// Note: Using {user_id} parameter for consistency with API standards
		r.Route("/user", func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)

			r.Get("/{user_id}", userHandler.GetUser)                 // Get user profile
			r.Patch("/{user_id}", userHandler.UpdateUser)            // Update user email
			r.Post("/{user_id}/upgrade", userHandler.UpgradeAccount) // Add email/password to mobile-only account
			r.Put("/{user_id}/password", userHandler.ChangePassword) // Change user password
			r.Delete("/{user_id}", userHandler.DeleteUser)           // Delete user account and all associated data
		})
	})

	return r
}
