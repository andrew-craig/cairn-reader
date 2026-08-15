package handlers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/middleware"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// paginationCursor is the decoded form of an opaque page cursor.
type paginationCursor struct {
	T  string `json:"t"`  // RFC3339Nano timestamp
	ID string `json:"id"` // UUID
}

func encodeCursor(addedAt time.Time, id uuid.UUID) string {
	c := paginationCursor{T: addedAt.UTC().Format(time.RFC3339Nano), ID: id.String()}
	data, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(data)
}

func decodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor encoding")
	}
	var c paginationCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339Nano, c.T)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor timestamp")
	}
	id, err := uuid.Parse(c.ID)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor id")
	}
	return t, id, nil
}

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
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own content", nil, "v1")
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	var cursorTime *time.Time
	var cursorID *uuid.UUID
	if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
		t, id, err := decodeCursor(cursorStr)
		if err != nil {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid cursor", nil, "v1")
			return
		}
		cursorTime = &t
		cursorID = &id
	}

	var status *string
	var isFavorite *bool

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if !middleware.ValidateStatus(statusStr) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, "Invalid status. Must be 'unread', 'reading', 'completed', or 'archived'", nil, "v1")
			return
		}
		status = &statusStr
	}

	if isFavStr := r.URL.Query().Get("is_favorite"); isFavStr != "" {
		switch isFavStr {
		case "true":
			fav := true
			isFavorite = &fav
		case "false":
			fav := false
			isFavorite = &fav
		}
	}

	// Fetch limit+1 to determine whether a next page exists.
	userContents, err := h.userContentRepo.ListByUserWithCursor(r.Context(), userID, status, isFavorite, limit+1, cursorTime, cursorID)
	if err != nil {
		slog.Error("failed to list user contents", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch user contents", nil, "v1")
		return
	}

	hasMore := len(userContents) > limit
	if hasMore {
		userContents = userContents[:limit]
	}

	// Batch-fetch all content records in one query to avoid N+1.
	contentIDs := make([]uuid.UUID, len(userContents))
	for i, uc := range userContents {
		contentIDs[i] = uc.ContentID
	}
	contentMap, err := h.contentRepo.GetByIDs(r.Context(), contentIDs)
	if err != nil {
		slog.Error("failed to fetch content details", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch user contents", nil, "v1")
		return
	}

	responses := make([]*dto.UserContentResponse, 0, len(userContents))
	for _, uc := range userContents {
		content := contentMap[uc.ContentID]
		if content == nil {
			continue
		}
		responses = append(responses, &dto.UserContentResponse{
			ID:             uc.ID,
			UserID:         uc.UserID,
			ContentID:      uc.ContentID,
			Status:         uc.Status,
			ScrollPosition: uc.ScrollPosition,
			IsFavorite:     uc.IsFavorite,
			AddedAt:        uc.AddedAt,
			UpdatedAt:      uc.UpdatedAt,
			Content:        contentToSummaryResponse(content),
		})
	}

	nextCursor := ""
	if hasMore && len(userContents) > 0 {
		last := userContents[len(userContents)-1]
		nextCursor = encodeCursor(last.AddedAt, last.ID)
	}

	api.WritePaginated(w, http.StatusOK, responses, api.PaginationInfo{
		Cursor:  nextCursor,
		HasMore: hasMore,
		Limit:   limit,
	}, "v1")
}

// GetUserContent handles GET /api/v1/content/user/:user_id/:content_id
// Returns the full content detail including cleaned_html.
func (h *UserContentHandler) GetUserContent(w http.ResponseWriter, r *http.Request) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own content", nil, "v1")
		return
	}

	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid content ID format", nil, "v1")
		return
	}

	userContent, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, contentID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch user content", nil, "v1")
		return
	}
	if userContent == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "User content not found", nil, "v1")
		return
	}

	content, err := h.contentRepo.GetByID(r.Context(), contentID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch content", nil, "v1")
		return
	}

	response := &dto.UserContentDetailResponse{
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

	api.WriteSuccess(w, http.StatusOK, response, "v1")
}

// AddContentToUser handles POST /api/v1/content/user/:user_id
// Supports two modes:
// 1. URL-based: Provide URL (optional Type/Title) for automatic detection and routing
// 2. Content-ID-based (legacy): Provide ContentID for pre-created content
func (h *UserContentHandler) AddContentToUser(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	// Get user ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	// Verify user owns the content
	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own content", nil, "v1")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSimpleRequestSize)

	var req dto.AddContentToUserRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
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
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Either 'url' or 'content_id' is required", nil, "v1")
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
		switch errMsg {
		case "already subscribed to this feed":
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "Already subscribed to this feed", nil, "v1")
		case "feed limit reached (max 100 feeds per user)":
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Feed limit reached (max 100 feeds per user)", nil, "v1")
		case "invalid feed URL or not a valid RSS/Atom feed":
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, "Invalid feed URL or not a valid RSS/Atom feed", nil, "v1")
		default:
			api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to subscribe to feed: "+err.Error(), nil, "v1")
		}
		return
	}

	// Build feed response
	response := &dto.AddFeedResponse{
		Type:   "feed",
		FeedID: subscription.FeedID,
		Subscription: dto.FeedSubscriptionDTO{
			ID:           subscription.SubscriptionID,
			UserID:       userID.String(),
			FeedID:       subscription.FeedID,
			FeedURL:      subscription.FeedURL,
			Title:        subscription.FeedTitle,
			SubscribedAt: subscription.SubscribedAt,
		},
	}

	api.WriteSuccess(w, http.StatusCreated, response, "v1")
}

// handlePageSubmission extracts content from a web page and adds to reading list
func (h *UserContentHandler) handlePageSubmission(w http.ResponseWriter, r *http.Request, userID uuid.UUID, url string, req *dto.AddContentToUserRequest) {
	// Create content from URL using ContentService
	content, err := h.contentService.CreateFromURL(r.Context(), url, "manual", nil, nil)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to extract content from URL: "+err.Error(), nil, "v1")
		return
	}

	// Check if user already has this content
	existing, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, content.ID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to check existing content: "+err.Error(), nil, "v1")
		return
	}
	if existing != nil {
		api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "User already has this content", nil, "v1")
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
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to add content to user: "+err.Error(), nil, "v1")
		return
	}

	// Build page response
	response := &dto.AddPageResponse{
		Type: "page",
		Content: &dto.UserContentDetailResponse{
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

	api.WriteSuccess(w, http.StatusCreated, response, "v1")
}

// handleContentIDBasedSubmission handles legacy content-ID-based submission
func (h *UserContentHandler) handleContentIDBasedSubmission(w http.ResponseWriter, r *http.Request, userID uuid.UUID, req *dto.AddContentToUserRequest) {
	contentID := *req.ContentID

	// Validate content ID exists
	_, err := h.contentRepo.GetByID(r.Context(), contentID)
	if err != nil {
		if err == sql.ErrNoRows {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Content not found", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch content: "+err.Error(), nil, "v1")
		return
	}

	// Check if user already has this content
	existing, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, contentID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to check existing content: "+err.Error(), nil, "v1")
		return
	}
	if existing != nil {
		api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "User already has this content", nil, "v1")
		return
	}

	// Set default status if not provided
	status := req.Status
	if status == "" {
		status = models.StatusUnread
	}

	// Validate status
	if !middleware.ValidateStatus(status) {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, "Invalid status. Must be 'unread', 'reading', 'completed', or 'archived'", nil, "v1")
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
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to add content to user: "+err.Error(), nil, "v1")
		return
	}

	// Get content details for response
	content, _ := h.contentRepo.GetByID(r.Context(), contentID)

	response := &dto.UserContentDetailResponse{
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

	api.WriteSuccess(w, http.StatusCreated, response, "v1")
}

// UpdateUserContent handles PATCH /api/v1/users/:user_id/contents/:content_id
func (h *UserContentHandler) UpdateUserContent(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	// Get user ID and content ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	// Verify user owns the content
	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own content", nil, "v1")
		return
	}

	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid content ID format", nil, "v1")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSimpleRequestSize)

	var req dto.UpdateUserContentRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
		return
	}

	// Get existing user-content
	userContent, err := h.userContentRepo.GetByUserAndContent(r.Context(), userID, contentID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch user-content: "+err.Error(), nil, "v1")
		return
	}
	if userContent == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "User content not found", nil, "v1")
		return
	}

	// Update metadata
	err = h.userContentRepo.UpdateMetadata(r.Context(), userContent.ID, req.Status, req.ScrollPosition, req.IsFavorite)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to update user-content: "+err.Error(), nil, "v1")
		return
	}

	// Get updated user-content
	userContent, err = h.userContentRepo.GetByID(r.Context(), userContent.ID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch updated user-content: "+err.Error(), nil, "v1")
		return
	}

	// Get content details for response
	content, _ := h.contentRepo.GetByID(r.Context(), contentID)

	response := &dto.UserContentDetailResponse{
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

	api.WriteSuccess(w, http.StatusOK, response, "v1")
}

// DeleteUserContent handles DELETE /api/v1/users/:user_id/contents/:content_id
func (h *UserContentHandler) DeleteUserContent(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	// Get user ID and content ID from URL
	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	// Verify user owns the content
	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own content", nil, "v1")
		return
	}

	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid content ID format", nil, "v1")
		return
	}

	// Delete user-content
	err = h.userContentRepo.Delete(r.Context(), userID, contentID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to delete user-content: "+err.Error(), nil, "v1")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchUserContents handles GET /api/v1/users/:user_id/contents/search
func (h *UserContentHandler) SearchUserContents(w http.ResponseWriter, r *http.Request) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own content", nil, "v1")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Search query 'q' is required", nil, "v1")
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	var cursorTime *time.Time
	var cursorID *uuid.UUID
	if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
		t, id, err := decodeCursor(cursorStr)
		if err != nil {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid cursor", nil, "v1")
			return
		}
		cursorTime = &t
		cursorID = &id
	}

	// Fetch limit+1 to determine whether a next page exists.
	userContents, err := h.userContentRepo.SearchWithCursor(r.Context(), userID, query, limit+1, cursorTime, cursorID)
	if err != nil {
		slog.Error("failed to search user contents", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to search user contents", nil, "v1")
		return
	}

	hasMore := len(userContents) > limit
	if hasMore {
		userContents = userContents[:limit]
	}

	// Batch-fetch all content records in one query to avoid N+1.
	searchContentIDs := make([]uuid.UUID, len(userContents))
	for i, uc := range userContents {
		searchContentIDs[i] = uc.ContentID
	}
	searchContentMap, err := h.contentRepo.GetByIDs(r.Context(), searchContentIDs)
	if err != nil {
		slog.Error("failed to fetch content details", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to search user contents", nil, "v1")
		return
	}

	responses := make([]*dto.UserContentResponse, 0, len(userContents))
	for _, uc := range userContents {
		content := searchContentMap[uc.ContentID]
		if content == nil {
			continue
		}
		responses = append(responses, &dto.UserContentResponse{
			ID:             uc.ID,
			UserID:         uc.UserID,
			ContentID:      uc.ContentID,
			Status:         uc.Status,
			ScrollPosition: uc.ScrollPosition,
			IsFavorite:     uc.IsFavorite,
			AddedAt:        uc.AddedAt,
			UpdatedAt:      uc.UpdatedAt,
			Content:        contentToSummaryResponse(content),
		})
	}

	nextCursor := ""
	if hasMore && len(userContents) > 0 {
		last := userContents[len(userContents)-1]
		nextCursor = encodeCursor(last.AddedAt, last.ID)
	}

	api.WritePaginated(w, http.StatusOK, responses, api.PaginationInfo{
		Cursor:  nextCursor,
		HasMore: hasMore,
		Limit:   limit,
	}, "v1")
}
