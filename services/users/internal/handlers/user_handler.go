package handlers

import (
	"errors"
	"net/http"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler handles user management HTTP requests
type UserHandler struct {
	userService services.UserService
}

// NewUserHandler creates a new user management handler
func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email *string `json:"email" binding:"omitempty,email"`
}

// UpgradeAccountRequest represents the request body for upgrading an account
type UpgradeAccountRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// GetUser handles GET /api/v1/user/{user_id}
// Retrieves user profile information for the authenticated user
func (h *UserHandler) GetUser(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDFromGinContext(c)
	if err != nil {
		api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Get user (service layer handles authorization)
	user, err := h.userService.GetUser(c.Request.Context(), requestingUserID, targetUserID)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.GinWriteError(c, http.StatusForbidden, api.ErrCodeForbidden, "you can only access your own user data", nil, "v1")
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			api.GinWriteError(c, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to retrieve user", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, user, "v1")
}

// UpdateUser handles PATCH /api/v1/user/{user_id}
// Updates user profile information
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDFromGinContext(c)
	if err != nil {
		api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Parse request body
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate that at least one field is being updated
	if req.Email == nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "no fields to update", nil, "v1")
		return
	}

	// Update user (service layer handles authorization)
	user, err := h.userService.UpdateUser(c.Request.Context(), requestingUserID, targetUserID, req.Email)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.GinWriteError(c, http.StatusForbidden, api.ErrCodeForbidden, "you can only update your own user data", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrAccountExists) {
			api.GinWriteError(c, http.StatusConflict, api.ErrCodeConflict, "an account with this email already exists", nil, "v1")
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			api.GinWriteError(c, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to update user", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, user, "v1")
}

// UpgradeAccount handles POST /api/v1/user/{user_id}/upgrade
// Upgrades a mobile-only account to a hybrid account with email/password
func (h *UserHandler) UpgradeAccount(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDFromGinContext(c)
	if err != nil {
		api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Parse request body
	var req UpgradeAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Upgrade account (service layer handles authorization)
	user, err := h.userService.UpgradeAccount(
		c.Request.Context(),
		requestingUserID,
		targetUserID,
		req.Email,
		req.Password,
	)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.GinWriteError(c, http.StatusForbidden, api.ErrCodeForbidden, "you can only upgrade your own account", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrNotMobileAccount) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "account is not a mobile-only account", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrAccountExists) {
			api.GinWriteError(c, http.StatusConflict, api.ErrCodeConflict, "an account with this email already exists", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrWeakPassword) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			api.GinWriteError(c, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to upgrade account", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, user, "v1")
}

// DeleteUser handles DELETE /api/v1/user/{user_id}
// Deletes a user account and all associated data
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDFromGinContext(c)
	if err != nil {
		api.GinWriteError(c, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.GinWriteError(c, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Delete user (service layer handles authorization)
	err = h.userService.DeleteUser(c.Request.Context(), requestingUserID, targetUserID)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.GinWriteError(c, http.StatusForbidden, api.ErrCodeForbidden, "you can only delete your own account", nil, "v1")
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			api.GinWriteError(c, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		api.GinWriteError(c, http.StatusInternalServerError, api.ErrCodeInternal, "failed to delete user", nil, "v1")
		return
	}

	api.GinWriteSuccess(c, http.StatusOK, gin.H{
		"message": "user account successfully deleted",
	}, "v1")
}
