package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/andrew-craig/cairn-read/content/internal/api/dto"
	"github.com/andrew-craig/cairn-read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn-read/content/internal/models"
	"github.com/andrew-craig/cairn-read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ContentHandler handles content-related HTTP requests
type ContentHandler struct {
	contentService service.ContentService
}

// NewContentHandler creates a new ContentHandler
func NewContentHandler(contentService service.ContentService) *ContentHandler {
	return &ContentHandler{
		contentService: contentService,
	}
}

// CreateContent handles POST /api/v1/contents
func (h *ContentHandler) CreateContent(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateContentRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate required fields
	if req.URL == "" {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "URL is required", nil)
		return
	}

	if req.SourceType == "" {
		req.SourceType = models.SourceTypeWeb // Default to web
	}

	if !middleware.ValidateSourceType(req.SourceType) {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "Invalid source_type. Must be 'rss' or 'web'", nil)
		return
	}

	// Create content
	var content *models.Content
	var err error

	if req.HTML != nil && *req.HTML != "" {
		// Create from HTML
		content, err = h.contentService.CreateFromHTML(r.Context(), req.URL, *req.HTML, req.SourceType, req.SourceFeedID, req.PublishedAt)
	} else {
		// Create from URL
		content, err = h.contentService.CreateFromURL(r.Context(), req.URL, req.SourceType, req.SourceFeedID, req.PublishedAt)
	}

	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "creation_failed", "Failed to create content: "+err.Error(), nil)
		return
	}

	// Convert to response DTO
	response := contentToResponse(content)
	middleware.WriteJSON(w, http.StatusCreated, response)
}

// UpdateContent handles PUT /api/v1/contents/:id
func (h *ContentHandler) UpdateContent(w http.ResponseWriter, r *http.Request) {
	// Get content ID from URL
	idStr := chi.URLParam(r, "id")
	contentID, err := uuid.Parse(idStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid content ID", nil)
		return
	}

	var req dto.UpdateContentRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate required fields
	if req.URL == "" || req.HTML == "" {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "URL and HTML are required", nil)
		return
	}

	// Update content
	content, err := h.contentService.UpdateContent(r.Context(), contentID, req.URL, req.HTML, req.PublishedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			middleware.WriteError(w, http.StatusNotFound, "not_found", "Content not found", nil)
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "update_failed", "Failed to update content: "+err.Error(), nil)
		return
	}

	// Convert to response DTO
	response := contentToResponse(content)
	middleware.WriteJSON(w, http.StatusOK, response)
}

// GetContent handles GET /api/v1/contents/:id
func (h *ContentHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	// Get content ID from URL
	idStr := chi.URLParam(r, "id")
	contentID, err := uuid.Parse(idStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid content ID", nil)
		return
	}

	// Get content
	content, err := h.contentService.GetByID(r.Context(), contentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			middleware.WriteError(w, http.StatusNotFound, "not_found", "Content not found", nil)
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, "fetch_failed", "Failed to fetch content: "+err.Error(), nil)
		return
	}

	// Convert to response DTO
	response := contentToResponse(content)
	middleware.WriteJSON(w, http.StatusOK, response)
}

// contentToResponse converts a models.Content to dto.ContentResponse
func contentToResponse(content *models.Content) *dto.ContentResponse {
	response := &dto.ContentResponse{
		ID:           content.ID,
		ContentHash:  content.ContentHash,
		CleanedHTML:  content.CleanedHTML,
		OriginalURL:  content.OriginalURL,
		CanonicalURL: content.CanonicalURL,
		Title:        content.Title,
		Author:       content.Author,
		PublishedAt:  content.PublishedAt,
		Description:  content.Description,
		SourceType:   content.SourceType,
		SourceFeedID: content.SourceFeedID,
		CreatedAt:    content.CreatedAt,
		UpdatedAt:    content.UpdatedAt,
	}

	// Convert pq.StringArray to []string
	if content.ImageURLs != nil {
		response.ImageURLs = []string(content.ImageURLs)
	}

	return response
}
