package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/andrew-craig/cairn/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn/services/read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn/services/read/content/internal/models"
	"github.com/andrew-craig/cairn/services/read/content/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// UserContentHandler handles user-content-related HTTP requests
type UserContentHandler struct {
	userContentRepo repository.UserContentRepository
	contentRepo     repository.ContentRepository
}

// NewUserContentHandler creates a new UserContentHandler
func NewUserContentHandler(userContentRepo repository.UserContentRepository, contentRepo repository.ContentRepository) *UserContentHandler {
	return &UserContentHandler{
		userContentRepo: userContentRepo,
		contentRepo:     contentRepo,
	}
}

// ListUserContents handles GET /api/v1/users/:user_id/contents
func (h *UserContentHandler) ListUserContents(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID", nil)
		return
	}

	// Parse query parameters
	limit := 20 // Default limit as per spec
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse filter parameters
	var status *string
	var isFavorite *bool

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if !middleware.ValidateStatus(statusStr) {
			middleware.WriteError(w, http.StatusBadRequest, "invalid_status", "Invalid status. Must be 'unread', 'read', or 'archived'", nil)
			return
		}
		status = &statusStr
	}

	if isFavStr := r.URL.Query().Get("is_favorite"); isFavStr != "" {
		if isFavStr == "true" {
			fav := true
			isFavorite = &fav
		} else if isFavStr == "false" {
			fav := false
			isFavorite = &fav
		}
	}

	// Get user contents with filters
	userContents, err := h.userContentRepo.ListByUserWithFilter(r.Context(), userID, status, isFavorite, limit, offset)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "fetch_failed", "Failed to fetch user contents: "+err.Error(), nil)
		return
	}

	// Get total count
	totalCount, err := h.userContentRepo.CountByUser(r.Context(), userID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "count_failed", "Failed to count user contents: "+err.Error(), nil)
		return
	}

	// Fetch content details for each user-content
	responses := make([]*dto.UserContentResponse, 0, len(userContents))
	for _, uc := range userContents {
		// Get content details
		content, err := h.contentRepo.GetByID(r.Context(), uc.ContentID)
		if err != nil {
			// Skip if content not found (shouldn't happen due to FK constraint)
			continue
		}

		response := &dto.UserContentResponse{
			ID:             uc.ID,
			UserID:         uc.UserID,
			ContentID:      uc.ContentID,
			Status:         uc.Status,
			ScrollPosition: uc.ScrollPosition,
			IsFavorite:     uc.IsFavorite,
			AddedAt:        uc.AddedAt,
			UpdatedAt:      uc.UpdatedAt,
			Content:        contentToResponse(content),
		}
		responses = append(responses, response)
	}

	// Build response
	listResponse := &dto.UserContentsListResponse{
		Contents:   responses,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	// Add next cursor if there are more items
	if offset+limit < int(totalCount) {
		nextOffset := offset + limit
		cursor := strconv.Itoa(nextOffset)
		listResponse.NextCursor = &cursor
	}

	middleware.WriteJSON(w, http.StatusOK, listResponse)
}

// AddContentToUser handles POST /api/v1/users/:user_id/contents
func (h *UserContentHandler) AddContentToUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID", nil)
		return
	}

	var req dto.AddContentToUserRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate content ID exists
	_, err = h.contentRepo.GetByID(r.Context(), req.ContentID)
	if err != nil {
		if err == sql.ErrNoRows {
			middleware.WriteError(w, http.StatusNotFound, "content_not_found", "Content not found", nil)
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "fetch_failed", "Failed to fetch content: "+err.Error(), nil)
		return
	}

	// Check if user already has this content
	existing, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, req.ContentID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "check_failed", "Failed to check existing content: "+err.Error(), nil)
		return
	}
	if existing != nil {
		middleware.WriteError(w, http.StatusConflict, "already_exists", "User already has this content", nil)
		return
	}

	// Set default status if not provided
	status := req.Status
	if status == "" {
		status = models.StatusUnread
	}

	// Validate status
	if !middleware.ValidateStatus(status) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_status", "Invalid status. Must be 'unread', 'read', or 'archived'", nil)
		return
	}

	// Create user-content relationship
	userContent := &models.UserContent{
		UserID:         userID,
		ContentID:      req.ContentID,
		Status:         status,
		ScrollPosition: req.ScrollPosition,
		IsFavorite:     req.IsFavorite,
	}

	if err := h.userContentRepo.Create(r.Context(), userContent); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "creation_failed", "Failed to add content to user: "+err.Error(), nil)
		return
	}

	// Get content details for response
	content, _ := h.contentRepo.GetByID(r.Context(), req.ContentID)

	response := &dto.UserContentResponse{
		ID:             userContent.ID,
		UserID:         userContent.UserID,
		ContentID:      userContent.ContentID,
		Status:         userContent.Status,
		ScrollPosition: userContent.ScrollPosition,
		IsFavorite:     userContent.IsFavorite,
		AddedAt:        userContent.AddedAt,
		UpdatedAt:      userContent.UpdatedAt,
		Content:        contentToResponse(content),
	}

	middleware.WriteJSON(w, http.StatusCreated, response)
}

// UpdateUserContent handles PATCH /api/v1/users/:user_id/contents/:content_id
func (h *UserContentHandler) UpdateUserContent(w http.ResponseWriter, r *http.Request) {
	// Get user ID and content ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID", nil)
		return
	}

	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_content_id", "Invalid content ID", nil)
		return
	}

	var req dto.UpdateUserContentRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate status if provided
	if req.Status != nil && !middleware.ValidateStatus(*req.Status) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_status", "Invalid status. Must be 'unread', 'read', or 'archived'", nil)
		return
	}

	// Get existing user-content
	userContent, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, contentID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "fetch_failed", "Failed to fetch user-content: "+err.Error(), nil)
		return
	}
	if userContent == nil {
		middleware.WriteError(w, http.StatusNotFound, "not_found", "User content not found", nil)
		return
	}

	// Update metadata
	err = h.userContentRepo.UpdateMetadata(r.Context(), userContent.ID, req.Status, req.ScrollPosition, req.IsFavorite)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "update_failed", "Failed to update user-content: "+err.Error(), nil)
		return
	}

	// Get updated user-content
	userContent, err = h.userContentRepo.GetByID(r.Context(), userContent.ID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "fetch_failed", "Failed to fetch updated user-content: "+err.Error(), nil)
		return
	}

	// Get content details for response
	content, _ := h.contentRepo.GetByID(r.Context(), contentID)

	response := &dto.UserContentResponse{
		ID:             userContent.ID,
		UserID:         userContent.UserID,
		ContentID:      userContent.ContentID,
		Status:         userContent.Status,
		ScrollPosition: userContent.ScrollPosition,
		IsFavorite:     userContent.IsFavorite,
		AddedAt:        userContent.AddedAt,
		UpdatedAt:      userContent.UpdatedAt,
		Content:        contentToResponse(content),
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

// DeleteUserContent handles DELETE /api/v1/users/:user_id/contents/:content_id
func (h *UserContentHandler) DeleteUserContent(w http.ResponseWriter, r *http.Request) {
	// Get user ID and content ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID", nil)
		return
	}

	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_content_id", "Invalid content ID", nil)
		return
	}

	// Delete user-content
	err = h.userContentRepo.Delete(r.Context(), userID, contentID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "delete_failed", "Failed to delete user-content: "+err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchUserContents handles GET /api/v1/users/:user_id/contents/search
func (h *UserContentHandler) SearchUserContents(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID", nil)
		return
	}

	// Get search query
	query := r.URL.Query().Get("q")
	if query == "" {
		middleware.WriteError(w, http.StatusBadRequest, "missing_query", "Search query 'q' is required", nil)
		return
	}

	// Parse pagination parameters
	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Search user contents
	userContents, err := h.userContentRepo.Search(r.Context(), userID, query, limit, offset)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "search_failed", "Failed to search user contents: "+err.Error(), nil)
		return
	}

	// Fetch content details for each user-content
	responses := make([]*dto.UserContentResponse, 0, len(userContents))
	for _, uc := range userContents {
		// Get content details
		content, err := h.contentRepo.GetByID(r.Context(), uc.ContentID)
		if err != nil {
			// Skip if content not found
			continue
		}

		response := &dto.UserContentResponse{
			ID:             uc.ID,
			UserID:         uc.UserID,
			ContentID:      uc.ContentID,
			Status:         uc.Status,
			ScrollPosition: uc.ScrollPosition,
			IsFavorite:     uc.IsFavorite,
			AddedAt:        uc.AddedAt,
			UpdatedAt:      uc.UpdatedAt,
			Content:        contentToResponse(content),
		}
		responses = append(responses, response)
	}

	middleware.WriteJSON(w, http.StatusOK, responses)
}
