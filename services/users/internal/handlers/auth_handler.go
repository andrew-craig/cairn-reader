package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/andrew-craig/cairn/services/users/internal/middleware"
	"github.com/andrew-craig/cairn/services/users/internal/services"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRequest represents the request body for email/password registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// RegisterMobileRequest represents the request body for mobile device registration
type RegisterMobileRequest struct {
	ExpoDeviceID string `json:"expo_device_id" binding:"required"`
}

// LoginRequest represents the request body for email/password login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginMobileRequest represents the request body for mobile device login
type LoginMobileRequest struct {
	ExpoDeviceID string `json:"expo_device_id" binding:"required"`
}

// RefreshRequest represents the request body for token refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest represents the request body for logout
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Register handles POST /auth/register
// Creates a new user account with email and password
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	// Register user
	authResp, err := h.authService.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrAccountExists) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error: "an account with this email already exists",
			})
			return
		}
		if errors.Is(err, services.ErrWeakPassword) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "invalid input provided",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to register user",
		})
		return
	}

	c.JSON(http.StatusCreated, authResp)
}

// RegisterMobile handles POST /auth/register/mobile
// Creates a new mobile-only account with Expo device ID
func (h *AuthHandler) RegisterMobile(c *gin.Context) {
	var req RegisterMobileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	// Get device info and IP address from request
	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	// Register mobile user
	authResp, err := h.authService.RegisterMobile(
		c.Request.Context(),
		req.ExpoDeviceID,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrAccountExists) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error: "an account with this device ID already exists",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "invalid input provided",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to register mobile user",
		})
		return
	}

	c.JSON(http.StatusCreated, authResp)
}

// Login handles POST /auth/login
// Authenticates a user with email and password
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	// Get device info and IP address from request
	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	// Authenticate user
	authResp, err := h.authService.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "invalid email or password",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "invalid input provided",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to authenticate user",
		})
		return
	}

	c.JSON(http.StatusOK, authResp)
}

// LoginMobile handles POST /auth/login/mobile
// Authenticates a user with Expo device ID
func (h *AuthHandler) LoginMobile(c *gin.Context) {
	var req LoginMobileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	// Get device info and IP address from request
	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	// Authenticate user
	authResp, err := h.authService.LoginMobile(
		c.Request.Context(),
		req.ExpoDeviceID,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "invalid device ID",
			})
			return
		}
		if errors.Is(err, services.ErrHybridAccountDeviceLogin) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "device login not allowed for accounts with email/password",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "invalid input provided",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to authenticate user",
		})
		return
	}

	c.JSON(http.StatusOK, authResp)
}

// Refresh handles POST /auth/refresh
// Validates a refresh token and issues a new access token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	// Get device info and IP address from request
	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	// Refresh access token
	authResp, err := h.authService.RefreshAccessToken(
		c.Request.Context(),
		req.RefreshToken,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "invalid or expired refresh token",
			})
			return
		}
		if strings.Contains(err.Error(), "token reuse detected") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "token reuse detected - all tokens have been revoked",
			})
			return
		}
		if strings.Contains(err.Error(), "expired") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "refresh token has expired",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "invalid input provided",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to refresh access token",
		})
		return
	}

	c.JSON(http.StatusOK, authResp)
}

// Logout handles POST /auth/logout
// Revokes a specific refresh token
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	// Revoke the token
	err := h.authService.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "invalid input provided",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to logout",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "successfully logged out",
	})
}

// LogoutAll handles POST /auth/logout-all
// Revokes all refresh tokens for the authenticated user
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "authentication required",
		})
		return
	}

	// Revoke all user tokens
	err = h.authService.LogoutAll(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to logout from all devices",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "successfully logged out from all devices",
	})
}
