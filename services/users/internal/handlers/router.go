package handlers

import (
	"time"

	"github.com/andrew-craig/cairn-core/user-service/internal/auth"
	"github.com/andrew-craig/cairn-core/user-service/internal/database"
	"github.com/andrew-craig/cairn-core/user-service/internal/middleware"
	"github.com/andrew-craig/cairn-core/user-service/internal/services"
	"github.com/gin-gonic/gin"
)

// RouterConfig holds dependencies needed to set up routes
type RouterConfig struct {
	DB                  *database.DB
	VaultClient         *auth.VaultClient
	AuthService         services.AuthService
	UserService         services.UserService
	JWTManager          *auth.JWTManager
	AuthRateLimit       int           // Requests per window for auth endpoints
	AuthRateLimitWindow time.Duration // Time window for auth rate limiting
}

// Router sets up the HTTP routes
func Router(config RouterConfig) *gin.Engine {
	router := gin.Default()

	// Apply global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.RequireHTTPS())
	router.Use(middleware.SecureHeadersRelaxed())

	// Initialize handlers
	healthHandler := NewHealthHandler(config.DB, config.VaultClient)
	authHandler := NewAuthHandler(config.AuthService)
	userHandler := NewUserHandler(config.UserService)

	// Health endpoints
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/ready", healthHandler.ReadyCheck)

	// Set default rate limit values if not provided
	authRateLimit := config.AuthRateLimit
	if authRateLimit == 0 {
		authRateLimit = 10 // Default: 10 requests per window
	}
	authRateLimitWindow := config.AuthRateLimitWindow
	if authRateLimitWindow == 0 {
		authRateLimitWindow = 1 * time.Minute // Default: 1 minute window
	}

	// Authentication endpoints with rate limiting
	auth := router.Group("/auth")
	auth.Use(middleware.RateLimitAuth(authRateLimit, authRateLimitWindow))
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/register/mobile", authHandler.RegisterMobile)
		auth.POST("/login", authHandler.Login)
		auth.POST("/login/mobile", authHandler.LoginMobile)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		// logout-all requires authentication
		auth.POST("/logout-all", middleware.JWTAuth(config.JWTManager), authHandler.LogoutAll)
	}

	// User management endpoints
	users := router.Group("/users")
	users.Use(middleware.JWTAuth(config.JWTManager)) // All user endpoints require authentication
	{
		users.GET("/:id", userHandler.GetUser)
		users.PATCH("/:id", userHandler.UpdateUser)
		users.POST("/:id/upgrade", userHandler.UpgradeAccount)
		users.DELETE("/:id", userHandler.DeleteUser)
	}

	return router
}
