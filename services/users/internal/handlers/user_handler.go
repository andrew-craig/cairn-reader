package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
	"github.com/cairn-app/cairn-reader/services/users/internal/validation"
	"github.com/go-chi/chi/v5"
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
	Email *string `json:"email"`
}

// ChangePasswordRequest represents the request body for changing a password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpgradeAccountRequest represents the request body for upgrading an account
type UpgradeAccountRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// GetUser handles GET /api/v1/user/{user_id}
// Retrieves user profile information for the authenticated user
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := chi.URLParam(r, "user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Get user (service layer handles authorization)
	user, err := h.userService.GetUser(r.Context(), requestingUserID, targetUserID)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "you can only access your own user data", nil, "v1")
			return
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to retrieve user", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, user, "v1")
}

// UpdateUser handles PATCH /api/v1/user/{user_id}
// Updates user profile information
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := chi.URLParam(r, "user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Parse request body
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate that at least one field is being updated
	if req.Email == nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "no fields to update", nil, "v1")
		return
	}

	// RFC 5322 email validation if provided
	if req.Email != nil {
		if err := validation.ValidateEmail(*req.Email); err != nil {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid email format", nil, "v1")
			return
		}
	}

	// Update user (service layer handles authorization)
	user, err := h.userService.UpdateUser(r.Context(), requestingUserID, targetUserID, req.Email)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "you can only update your own user data", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrAccountExists) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "an account with this email already exists", nil, "v1")
			return
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to update user", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, user, "v1")
}

// UpgradeAccount handles POST /api/v1/user/{user_id}/upgrade
// Upgrades a mobile-only account to a hybrid account with email/password
func (h *UserHandler) UpgradeAccount(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := chi.URLParam(r, "user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Parse request body
	var req UpgradeAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	// Upgrade account (service layer handles authorization)
	user, err := h.userService.UpgradeAccount(
		r.Context(),
		requestingUserID,
		targetUserID,
		req.Email,
		req.Password,
	)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "you can only upgrade your own account", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrNotMobileAccount) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "account is not a mobile-only account", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrAccountExists) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "an account with this email already exists", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrWeakPassword) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
			return
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to upgrade account", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, user, "v1")
}

// ChangePassword handles PUT /api/v1/user/{user_id}/password
// Changes the authenticated user's password
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := chi.URLParam(r, "user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Parse request body
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request: "+err.Error(), nil, "v1")
		return
	}

	// Validate required fields
	if req.CurrentPassword == "" || req.NewPassword == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "current_password and new_password are required", nil, "v1")
		return
	}

	// Change password (service layer handles authorization and validation)
	err = h.userService.ChangePassword(r.Context(), requestingUserID, targetUserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "you can only change your own password", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrIncorrectPassword) {
			api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "current password is incorrect", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrNoPassword) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "account does not have a password", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrWeakPassword) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
			return
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		if errors.Is(err, services.ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to change password", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]string{
		"message": "password changed successfully",
	}, "v1")
}

// DeleteUser handles DELETE /api/v1/user/{user_id}
// Deletes a user account and all associated data
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	requestingUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "authentication required", nil, "v1")
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := chi.URLParam(r, "user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid user ID format", nil, "v1")
		return
	}

	// Delete user (service layer handles authorization)
	err = h.userService.DeleteUser(r.Context(), requestingUserID, targetUserID)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "you can only delete your own account", nil, "v1")
			return
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to delete user", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]string{
		"message": "user account successfully deleted",
	}, "v1")
}
