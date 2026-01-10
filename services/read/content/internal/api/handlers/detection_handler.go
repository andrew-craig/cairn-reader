package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/middleware"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
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
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
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

	api.WriteSuccess(w, http.StatusOK, response, "v1")
}
