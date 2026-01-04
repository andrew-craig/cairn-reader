package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/andrew-craig/cairn/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn/services/read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn/services/read/content/internal/models"
	"github.com/andrew-craig/cairn/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// UserContentHandler handles user-content-related HTTP requests
type UserContentHandler struct {
	userContentRepo repository.UserContentRepository
	contentRepo     repository.ContentRepository
	contentService  service.ContentService
	urlDetector     service.URLDetector
	ingestRSSClient *service.IngestRSSClient
}

// NewUserContentHandler creates a new UserContentHandler
func NewUserContentHandler(
	userContentRepo repository.UserContentRepository,
	contentRepo repository.ContentRepository,
	contentService service.ContentService,
	urlDetector service.URLDetector,
	ingestRSSClient *service.IngestRSSClient,
) *UserContentHandler {
	return &UserContentHandler{
		userContentRepo: userContentRepo,
		contentRepo:     contentRepo,
		contentService:  contentService,
		urlDetector:     urlDetector,
		ingestRSSClient: ingestRSSClient,
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

// AddContentToUser handles POST /api/v1/content/user/:user_id
// Supports two modes:
// 1. URL-based: Provide URL (optional Type/Title) for automatic detection and routing
// 2. Content-ID-based (legacy): Provide ContentID for pre-created content
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

	// Determine which mode: URL-based or ContentID-based
	if req.URL != nil && *req.URL != "" {
		// NEW FLOW: URL-based submission
		h.handleURLBasedSubmission(w, r, userID, &req)
	} else if req.ContentID != nil {
		// LEGACY FLOW: Content-ID-based submission
		h.handleContentIDBasedSubmission(w, r, userID, &req)
	} else {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Either 'url' or 'content_id' is required", nil)
	}
}

// handleURLBasedSubmission handles URL-based content submission with auto-routing
func (h *UserContentHandler) handleURLBasedSubmission(w http.ResponseWriter, r *http.Request, userID uuid.UUID, req *dto.AddContentToUserRequest) {
	url := *req.URL

	// Determine type: use provided type or detect
	var urlType service.URLType
	if req.Type != nil && *req.Type != "" {
		urlType = service.URLType(*req.Type)
	} else {
		// No type provided, run detection
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		result, err := h.urlDetector.DetectURL(ctx, url)
		if err != nil || result.Type == service.URLTypeUnknown {
			// Default to page if detection fails
			urlType = service.URLTypePage
		} else {
			urlType = result.Type
		}
	}

	// Route based on detected type
	switch urlType {
	case service.URLTypeFeed:
		h.handleFeedSubmission(w, r, userID, url)
	case service.URLTypePage, service.URLTypeUnknown:
		h.handlePageSubmission(w, r, userID, url, req)
	default:
		h.handlePageSubmission(w, r, userID, url, req)
	}
}

// handleFeedSubmission subscribes the user to an RSS feed
func (h *UserContentHandler) handleFeedSubmission(w http.ResponseWriter, r *http.Request, userID uuid.UUID, feedURL string) {
	// Call Ingest RSS service to subscribe user to feed
	subscription, err := h.ingestRSSClient.SubscribeUserToFeed(r.Context(), userID.String(), feedURL)
	if err != nil {
		// Map errors to user-friendly messages
		errMsg := err.Error()
		if errMsg == "already subscribed to this feed" {
			middleware.WriteError(w, http.StatusConflict, "already_subscribed", "Already subscribed to this feed", nil)
		} else if errMsg == "feed limit reached (max 100 feeds per user)" {
			middleware.WriteError(w, http.StatusBadRequest, "feed_limit_reached", "Feed limit reached (max 100 feeds per user)", nil)
		} else if errMsg == "invalid feed URL or not a valid RSS/Atom feed" {
			middleware.WriteError(w, http.StatusBadRequest, "invalid_feed", "Invalid feed URL or not a valid RSS/Atom feed", nil)
		} else {
			middleware.WriteError(w, http.StatusInternalServerError, "feed_subscription_failed", "Failed to subscribe to feed: "+err.Error(), nil)
		}
		return
	}

	// Build feed response
	response := &dto.AddFeedResponse{
		Type:   "feed",
		FeedID: subscription.Feed.ID,
		Subscription: dto.FeedSubscriptionDTO{
			ID:           subscription.Subscription.ID,
			UserID:       subscription.Subscription.UserID,
			FeedID:       subscription.Subscription.FeedID,
			FeedURL:      subscription.Feed.FeedURL,
			Title:        subscription.Feed.Title,
			SubscribedAt: subscription.Subscription.SubscribedAt,
		},
	}

	middleware.WriteJSON(w, http.StatusCreated, response)
}

// handlePageSubmission extracts content from a web page and adds to reading list
func (h *UserContentHandler) handlePageSubmission(w http.ResponseWriter, r *http.Request, userID uuid.UUID, url string, req *dto.AddContentToUserRequest) {
	// Create content from URL using ContentService
	content, err := h.contentService.CreateFromURL(r.Context(), url, "manual", nil, nil)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "content_extraction_failed", "Failed to extract content from URL: "+err.Error(), nil)
		return
	}

	// Check if user already has this content
	existing, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, content.ID)
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

	// Create user-content relationship
	userContent := &models.UserContent{
		UserID:         userID,
		ContentID:      content.ID,
		Status:         status,
		ScrollPosition: req.ScrollPosition,
		IsFavorite:     req.IsFavorite,
	}

	if err := h.userContentRepo.Create(r.Context(), userContent); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "creation_failed", "Failed to add content to user: "+err.Error(), nil)
		return
	}

	// Build page response
	response := &dto.AddPageResponse{
		Type: "page",
		Content: &dto.UserContentResponse{
			ID:             userContent.ID,
			UserID:         userContent.UserID,
			ContentID:      userContent.ContentID,
			Status:         userContent.Status,
			ScrollPosition: userContent.ScrollPosition,
			IsFavorite:     userContent.IsFavorite,
			AddedAt:        userContent.AddedAt,
			UpdatedAt:      userContent.UpdatedAt,
			Content:        contentToResponse(content),
		},
	}

	middleware.WriteJSON(w, http.StatusCreated, response)
}

// handleContentIDBasedSubmission handles legacy content-ID-based submission
func (h *UserContentHandler) handleContentIDBasedSubmission(w http.ResponseWriter, r *http.Request, userID uuid.UUID, req *dto.AddContentToUserRequest) {
	contentID := *req.ContentID

	// Validate content ID exists
	_, err := h.contentRepo.GetByID(r.Context(), contentID)
	if err != nil {
		if err == sql.ErrNoRows {
			middleware.WriteError(w, http.StatusNotFound, "content_not_found", "Content not found", nil)
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "fetch_failed", "Failed to fetch content: "+err.Error(), nil)
		return
	}

	// Check if user already has this content
	existing, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, contentID)
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
		ContentID:      contentID,
		Status:         status,
		ScrollPosition: req.ScrollPosition,
		IsFavorite:     req.IsFavorite,
	}

	if err := h.userContentRepo.Create(r.Context(), userContent); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "creation_failed", "Failed to add content to user: "+err.Error(), nil)
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
