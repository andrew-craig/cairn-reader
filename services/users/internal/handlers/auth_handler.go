package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
	"github.com/cairn-app/cairn-reader/services/users/internal/validation"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService              services.AuthService
	emailVerificationService services.EmailVerificationService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService services.AuthService, emailVerificationService services.EmailVerificationService) *AuthHandler {
	return &AuthHandler{
		authService:              authService,
		emailVerificationService: emailVerificationService,
	}
}

// RegisterRequest represents the request body for email/password registration
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterMobileRequest represents the request body for mobile device registration
type RegisterMobileRequest struct {
	ExpoDeviceID string `json:"expo_device_id"`
}

// LoginRequest represents the request body for email/password login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginMobileRequest represents the request body for mobile device login
type LoginMobileRequest struct {
	ExpoDeviceID string `json:"expo_device_id"`
}

// RefreshRequest represents the request body for token refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest represents the request body for logout
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Register handles POST /auth/register
// Creates a new user account with email and password
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "email and password are required", nil, "v1")
		return
	}

	// RFC 5322 email validation
	if err := validation.ValidateEmail(req.Email); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid email format", nil, "v1")
		return
	}

	// Password minimum length
	if len(req.Password) < 8 {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "password must be at least 8 characters", nil, "v1")
		return
	}

	// Register user
	authResp, err := h.authService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrAccountExists) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "an account with this email already exists", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrWeakPassword) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to register user", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusCreated, authResp, "v1")
}

// RegisterMobile handles POST /auth/register/mobile
// Creates a new mobile-only account with Expo device ID
func (h *AuthHandler) RegisterMobile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req RegisterMobileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate required fields
	if req.ExpoDeviceID == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "expo_device_id is required", nil, "v1")
		return
	}

	// Get device info and IP address from request
	deviceInfo := r.Header.Get("User-Agent")
	ipAddress := getClientIP(r)

	// Register mobile user
	authResp, err := h.authService.RegisterMobile(
		r.Context(),
		req.ExpoDeviceID,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrAccountExists) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "an account with this device ID already exists", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to register mobile user", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusCreated, authResp, "v1")
}

// Login handles POST /auth/login
// Authenticates a user with email and password
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "email and password are required", nil, "v1")
		return
	}

	// Get device info and IP address from request
	deviceInfo := r.Header.Get("User-Agent")
	ipAddress := getClientIP(r)

	// Authenticate user
	authResp, err := h.authService.Login(
		r.Context(),
		req.Email,
		req.Password,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrAccountLocked) {
			api.WriteError(w, http.StatusTooManyRequests, api.ErrCodeTooManyRequests, err.Error(), nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidCredentials) {
			api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid email or password", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to authenticate user", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, authResp, "v1")
}

// LoginMobile handles POST /auth/login/mobile
// Authenticates a user with Expo device ID
func (h *AuthHandler) LoginMobile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req LoginMobileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate required fields
	if req.ExpoDeviceID == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "expo_device_id is required", nil, "v1")
		return
	}

	// Get device info and IP address from request
	deviceInfo := r.Header.Get("User-Agent")
	ipAddress := getClientIP(r)

	// Authenticate user
	authResp, err := h.authService.LoginMobile(
		r.Context(),
		req.ExpoDeviceID,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrAccountLocked) {
			api.WriteError(w, http.StatusTooManyRequests, api.ErrCodeTooManyRequests, err.Error(), nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidCredentials) {
			api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid device ID", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrHybridAccountDeviceLogin) {
			api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "device login not allowed for accounts with email/password", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to authenticate user", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, authResp, "v1")
}

// Refresh handles POST /auth/refresh
// Validates a refresh token and issues a new access token
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ipAddress := getClientIP(r)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			slog.Warn("refresh request: body too large", slog.String("ip", ipAddress))
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		slog.Error("refresh request: failed to parse JSON",
			slog.String("error", err.Error()),
			slog.String("ip", ipAddress),
		)
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate required fields
	if req.RefreshToken == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "refresh_token is required", nil, "v1")
		return
	}

	// Get device info and IP address from request
	deviceInfo := r.Header.Get("User-Agent")
	ipAddress := getClientIP(r)

	// Get logger for structured logging
	logger := logging.FromContext(r.Context())
	requestID := logging.GetRequestIDFromContext(r.Context())

	logger.Info("refresh request: processing",
		slog.String("request_id", requestID),
		slog.String("client_ip", ipAddress),
	)

	// Refresh access token
	authResp, err := h.authService.RefreshAccessToken(
		r.Context(),
		req.RefreshToken,
		deviceInfo,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			logger.Warn("refresh token invalid or not found",
				slog.String("request_id", requestID),
				slog.String("client_ip", ipAddress),
				slog.String("error", err.Error()),
			)
			api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid or expired refresh token", nil, "v1")
			return
		}
		if strings.Contains(err.Error(), "token reuse detected") {
			logger.Warn("refresh token reuse detected - potential security issue",
				slog.String("request_id", requestID),
				slog.String("client_ip", ipAddress),
				slog.String("error", err.Error()),
			)
			api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "token reuse detected - all tokens have been revoked", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrRefreshTokenExpired) {
			logger.Info("refresh token expired",
				slog.String("request_id", requestID),
				slog.String("client_ip", ipAddress),
			)
			api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "refresh token has expired", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			logger.Warn("refresh request: invalid input",
				slog.String("request_id", requestID),
				slog.String("client_ip", ipAddress),
				slog.String("error", err.Error()),
			)
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		logger.Error("failed to refresh access token",
			slog.String("request_id", requestID),
			slog.String("client_ip", ipAddress),
			slog.String("error", err.Error()),
		)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to refresh access token", nil, "v1")
		return
	}

	logger.Info("refresh request: success",
		slog.String("request_id", requestID),
		slog.String("user_id", authResp.User.ID.String()),
		slog.String("client_ip", ipAddress),
	)

	api.WriteSuccess(w, http.StatusOK, authResp, "v1")
}

// Logout handles POST /auth/logout
// Revokes a specific refresh token
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate required fields
	if req.RefreshToken == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "refresh_token is required", nil, "v1")
		return
	}

	// Revoke the token
	err := h.authService.Logout(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to logout", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]string{
		"message": "successfully logged out",
	}, "v1")
}

// LogoutAll handles POST /auth/logout-all
// Revokes all refresh tokens for the authenticated user
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by JWT middleware)
	userID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Revoke all user tokens
	err = h.authService.LogoutAll(r.Context(), userID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to logout from all devices", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]string{
		"message": "successfully logged out from all devices",
	}, "v1")
}

// VerifyEmailRequest represents the request body for email verification
type VerifyEmailRequest struct {
	Token string `json:"token"`
}

// ResendVerificationRequest represents the request body for resending verification email
type ResendVerificationRequest struct {
	UserID string `json:"user_id"`
}

// VerifyEmail handles POST /auth/verify-email and GET /auth/verify-email?token=...
// Accepts a verification token and marks the user's email as verified.
// Supports both query parameter (for email links) and JSON body.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var token string
	if q := r.URL.Query().Get("token"); q != "" {
		token = q
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

		var req VerifyEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
				return
			}
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
			return
		}
		token = req.Token
	}

	if token == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "token is required", nil, "v1")
		return
	}

	user, err := h.emailVerificationService.VerifyEmail(r.Context(), token)
	if err != nil {
		if errors.Is(err, services.ErrInvalidVerificationToken) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid or expired verification token", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to verify email", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, user, "v1")
}

// ResendVerification handles POST /auth/resend-verification
// Generates a new verification token and logs the verification URL.
// Requires authentication — users can only resend verification for their own account.
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	if err := h.emailVerificationService.SendVerificationEmail(r.Context(), userID); err != nil {
		if errors.Is(err, services.ErrEmailAlreadyVerified) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "email is already verified", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "account does not have an email address", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to resend verification email", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]string{
		"message": "verification email sent",
	}, "v1")
}

// getClientIP extracts the client IP address from the request
// It checks X-Forwarded-For and X-Real-IP headers first (for reverse proxy setups)
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (comma-separated list, first is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is "IP:port", so strip the port
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
