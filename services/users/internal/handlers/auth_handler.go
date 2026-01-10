package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
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
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Register user
	authResp, err := h.authService.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrAccountExists) {
			api.GinWriteError(c, http.StatusConflict, api.ErrCodeConflict, "an account with this email already exists", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrWeakPassword) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to register user", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusCreated, authResp, "v1")
}

// RegisterMobile handles POST /auth/register/mobile
// Creates a new mobile-only account with Expo device ID
func (h *AuthHandler) RegisterMobile(c *gin.Context) {
	var req RegisterMobileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
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
			api.GinWriteError(c, http.StatusConflict, api.ErrCodeConflict, "an account with this device ID already exists", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to register mobile user", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusCreated, authResp, "v1")
}

// Login handles POST /auth/login
// Authenticates a user with email and password
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
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
			api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid email or password", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to authenticate user", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, authResp, "v1")
}

// LoginMobile handles POST /auth/login/mobile
// Authenticates a user with Expo device ID
func (h *AuthHandler) LoginMobile(c *gin.Context) {
	var req LoginMobileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
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
			api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid device ID", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrHybridAccountDeviceLogin) {
			api.GinWriteError(c, http.StatusForbidden, api.ErrCodeForbidden, "device login not allowed for accounts with email/password", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to authenticate user", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, authResp, "v1")
}

// Refresh handles POST /auth/refresh
// Validates a refresh token and issues a new access token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
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
			api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid or expired refresh token", nil, "v1")
			return
		}
		if strings.Contains(err.Error(), "token reuse detected") {
			api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "token reuse detected - all tokens have been revoked", nil, "v1")
			return
		}
		if strings.Contains(err.Error(), "expired") {
			api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "refresh token has expired", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to refresh access token", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, authResp, "v1")
}

// Logout handles POST /auth/logout
// Revokes a specific refresh token
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Revoke the token
	err := h.authService.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to logout", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, gin.H{
		"message": "successfully logged out",
	}, "v1")
}

// LogoutAll handles POST /auth/logout-all
// Revokes all refresh tokens for the authenticated user
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userID, err := auth.GetUserIDFromGinContext(c)
	if err != nil {
		api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Revoke all user tokens
	err = h.authService.LogoutAll(c.Request.Context(), userID)
	if err != nil {
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to logout from all devices", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, gin.H{
		"message": "successfully logged out from all devices",
	}, "v1")
}
