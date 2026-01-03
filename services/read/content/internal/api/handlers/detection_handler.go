package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn/services/read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn/services/read/content/internal/service"
)

// DetectionHandler handles URL detection requests
type DetectionHandler struct {
	urlDetector service.URLDetector
}

// NewDetectionHandler creates a new DetectionHandler
func NewDetectionHandler(urlDetector service.URLDetector) *DetectionHandler {
	return &DetectionHandler{
		urlDetector: urlDetector,
	}
}

// DetectURL handles POST /api/v1/content/detect
func (h *DetectionHandler) DetectURL(w http.ResponseWriter, r *http.Request) {
	var req dto.DetectURLRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	// Validate URL
	if req.URL == "" {
		middleware.WriteError(w, http.StatusBadRequest, "validation_error", "URL is required", nil)
		return
	}

	// Create context with 10s timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Detect URL type
	result, err := h.urlDetector.DetectURL(ctx, req.URL)
	if err != nil {
		// On error, return unknown
		result = &service.URLDetectionResult{
			URL:   req.URL,
			Type:  service.URLTypeUnknown,
			Title: nil,
		}
	}

	// Convert to response DTO
	response := &dto.DetectURLResponse{
		URL:   result.URL,
		Type:  string(result.Type),
		Title: result.Title,
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}
