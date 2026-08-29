package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/andrew-craig/cairn-reader/services/users/internal/services"
	"github.com/andrew-craig/cairn-reader/services/users/internal/validation"
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
		writeServiceError(w, err, "failed to retrieve user")
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

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	// Parse request body
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
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
		writeServiceError(w, err, "failed to update user")
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

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	// Parse request body
	var req UpgradeAccountRequest
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

	// Upgrade account (service layer handles authorization)
	user, err := h.userService.UpgradeAccount(
		r.Context(),
		requestingUserID,
		targetUserID,
		req.Email,
		req.Password,
	)
	if err != nil {
		writeServiceError(w, err, "failed to upgrade account")
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

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	// Parse request body
	var req ChangePasswordRequest
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
	if req.CurrentPassword == "" || req.NewPassword == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "current_password and new_password are required", nil, "v1")
		return
	}

	// Change password (service layer handles authorization and validation)
	err = h.userService.ChangePassword(r.Context(), requestingUserID, targetUserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		writeServiceError(w, err, "failed to change password")
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
		writeServiceError(w, err, "failed to delete user")
		return
	}

	api.WriteSuccess(w, http.StatusOK, map[string]string{
		"message": "user account successfully deleted",
	}, "v1")
}
