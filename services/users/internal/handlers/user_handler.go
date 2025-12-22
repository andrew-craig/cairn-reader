package handlers

import (
	"errors"
	"net/http"

	"github.com/andrew-craig/cairn-core/user-service/internal/database"
	"github.com/andrew-craig/cairn-core/user-service/internal/middleware"
	"github.com/andrew-craig/cairn-core/user-service/internal/services"
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

// GetUser handles GET /users/{id}
// Retrieves user profile information for the authenticated user
func (h *UserHandler) GetUser(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "authentication required",
		})
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid user ID format",
		})
		return
	}

	// Get user (service layer handles authorization)
	user, err := h.userService.GetUser(c.Request.Context(), requestingUserID, targetUserID)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "you can only access your own user data",
			})
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "user not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to retrieve user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles PATCH /users/{id}
// Updates user profile information
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "authentication required",
		})
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid user ID format",
		})
		return
	}

	// Parse request body
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	// Validate that at least one field is being updated
	if req.Email == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "no fields to update",
		})
		return
	}

	// Update user (service layer handles authorization)
	user, err := h.userService.UpdateUser(c.Request.Context(), requestingUserID, targetUserID, req.Email)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "you can only update your own user data",
			})
			return
		}
		if errors.Is(err, services.ErrAccountExists) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error: "an account with this email already exists",
			})
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "user not found",
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
			Error: "failed to update user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpgradeAccount handles POST /users/{id}/upgrade
// Upgrades a mobile-only account to a hybrid account with email/password
func (h *UserHandler) UpgradeAccount(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "authentication required",
		})
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid user ID format",
		})
		return
	}

	// Parse request body
	var req UpgradeAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request: " + err.Error(),
		})
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
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "you can only upgrade your own account",
			})
			return
		}
		if errors.Is(err, services.ErrNotMobileAccount) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "account is not a mobile-only account",
			})
			return
		}
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
		if errors.Is(err, database.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "user not found",
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
			Error: "failed to upgrade account",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser handles DELETE /users/{id}
// Deletes a user account and all associated data
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// Get authenticated user ID from context
	requestingUserID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "authentication required",
		})
		return
	}

	// Parse target user ID from URL parameter
	targetUserIDStr := c.Param("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid user ID format",
		})
		return
	}

	// Delete user (service layer handles authorization)
	err = h.userService.DeleteUser(c.Request.Context(), requestingUserID, targetUserID)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "you can only delete your own account",
			})
			return
		}
		if errors.Is(err, database.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "user not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to delete user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user account successfully deleted",
	})
}
