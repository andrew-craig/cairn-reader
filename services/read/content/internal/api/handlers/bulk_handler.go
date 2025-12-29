package handlers

import (
	"net/http"

	"github.com/andrew-craig/cairn/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn/services/read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn/services/read/content/internal/models"
	"github.com/andrew-craig/cairn/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn/services/read/content/internal/service"
	"github.com/google/uuid"
)

// BulkHandler handles bulk operation HTTP requests
type BulkHandler struct {
	contentService      service.ContentService
	userContentRepo     repository.UserContentRepository
	contentRepo         repository.ContentRepository
}

// NewBulkHandler creates a new BulkHandler
func NewBulkHandler(contentService service.ContentService, userContentRepo repository.UserContentRepository, contentRepo repository.ContentRepository) *BulkHandler {
	return &BulkHandler{
		contentService:  contentService,
		userContentRepo: userContentRepo,
		contentRepo:     contentRepo,
	}
}

// BulkCreateContent handles POST /api/v1/contents/bulk
func (h *BulkHandler) BulkCreateContent(w http.ResponseWriter, r *http.Request) {
	var req dto.BulkCreateContentRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate max 100 items
	if len(req.Contents) == 0 {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Contents array cannot be empty", nil)
		return
	}
	if len(req.Contents) > 100 {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Maximum 100 items allowed per bulk request", nil)
		return
	}

	// Convert DTOs to service types
	serviceItems := make([]service.BulkContentItem, len(req.Contents))
	for i, item := range req.Contents {
		// Validate required fields
		if item.URL == "" || item.HTML == "" {
			middleware.WriteError(w, http.StatusBadRequest, "validation_error", "URL and HTML are required for all items", nil)
			return
		}

		// Default source type to RSS if not specified
		if item.SourceType == "" {
			item.SourceType = models.SourceTypeRSS
		}

		if !middleware.ValidateSourceType(item.SourceType) {
			middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Invalid source_type. Must be 'rss' or 'web'", nil)
			return
		}

		serviceItems[i] = service.BulkContentItem{
			URL:          item.URL,
			HTML:         item.HTML,
			SourceType:   item.SourceType,
			SourceFeedID: item.SourceFeedID,
			PublishedAt:  item.PublishedAt,
		}
	}

	// Process bulk creation
	contents, errors, err := h.contentService.BulkCreateFromHTML(r.Context(), serviceItems)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "bulk_creation_failed", "Failed to bulk create contents: "+err.Error(), nil)
		return
	}

	// Build response
	response := &dto.BulkCreateContentResponse{
		Created:  make([]*dto.ContentResponse, 0),
		Existing: make([]*dto.ContentResponse, 0),
		Failed:   make([]dto.BulkFailedItem, 0),
	}

	// Separate created vs existing based on whether they were just created or already existed
	for _, content := range contents {
		contentResp := contentToResponse(content)
		// For simplicity, we'll put all in created. In a more sophisticated implementation,
		// we could track which ones were already existing
		response.Created = append(response.Created, contentResp)
	}

	// Add failed items
	for _, bulkErr := range errors {
		response.Failed = append(response.Failed, dto.BulkFailedItem{
			Index:   bulkErr.Index,
			URL:     bulkErr.URL,
			Error:   "processing_error",
			Message: bulkErr.Message,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

// CheckDuplicates handles POST /api/v1/contents/check-duplicates
func (h *BulkHandler) CheckDuplicates(w http.ResponseWriter, r *http.Request) {
	var req dto.CheckDuplicatesRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate max 100 items
	if len(req.Items) == 0 {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Items array cannot be empty", nil)
		return
	}
	if len(req.Items) > 100 {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Maximum 100 items allowed per request", nil)
		return
	}

	// Convert DTOs to service types
	serviceItems := make([]service.DuplicateCheckItem, len(req.Items))
	for i, item := range req.Items {
		serviceItems[i] = service.DuplicateCheckItem{
			ContentHash:  item.ContentHash,
			SourceFeedID: item.SourceFeedID,
		}
	}

	// Check duplicates
	existingContents, err := h.contentService.CheckDuplicates(r.Context(), serviceItems)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "check_failed", "Failed to check duplicates: "+err.Error(), nil)
		return
	}

	// Build response
	response := &dto.CheckDuplicatesResponse{
		Results: make([]dto.DuplicateCheckResult, 0),
	}

	for _, item := range req.Items {
		result := dto.DuplicateCheckResult{
			ContentHash: item.ContentHash,
			Exists:      false,
		}

		if content, found := existingContents[item.ContentHash]; found {
			result.Exists = true
			result.ContentID = &content.ID
			result.Content = contentToResponse(content)
		}

		response.Results = append(response.Results, result)
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

// BulkAddToUsers handles POST /api/v1/users/bulk/contents
func (h *BulkHandler) BulkAddToUsers(w http.ResponseWriter, r *http.Request) {
	var req dto.BulkAddToUsersRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate max 100 items
	if len(req.Items) == 0 {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Items array cannot be empty", nil)
		return
	}
	if len(req.Items) > 100 {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Maximum 100 items allowed per request", nil)
		return
	}

	// Validate and prepare user-content records
	userContents := make([]*models.UserContent, len(req.Items))
	for i, item := range req.Items {
		// Validate status if provided
		status := item.Status
		if status == "" {
			status = models.StatusUnread // Default to unread
		}
		if !models.ValidateStatus(status) {
			middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Invalid status. Must be 'unread', 'read', or 'archived'", nil)
			return
		}

		userContents[i] = &models.UserContent{
			ID:             uuid.New(),
			UserID:         item.UserID,
			ContentID:      item.ContentID,
			Status:         status,
			ScrollPosition: 0,
			IsFavorite:     false,
		}
	}

	// Bulk create user-content relationships
	err := h.userContentRepo.BulkCreate(r.Context(), userContents)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "bulk_add_failed", "Failed to bulk add contents to users: "+err.Error(), nil)
		return
	}

	// Build response
	response := &dto.BulkAddToUsersResponse{
		Succeeded: make([]uuid.UUID, 0),
		Failed:    make([]dto.BulkFailedItem, 0),
	}

	// All succeeded (conflicts are ignored)
	for _, uc := range userContents {
		response.Succeeded = append(response.Succeeded, uc.ID)
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}
