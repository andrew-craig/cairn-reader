package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// FeedbackHandler handles extraction feedback HTTP requests
type FeedbackHandler struct {
	feedbackService *service.FeedbackService
	feedbackRepo    repository.ContentFeedbackRepository
}

// NewFeedbackHandler creates a new FeedbackHandler
func NewFeedbackHandler(feedbackService *service.FeedbackService, feedbackRepo repository.ContentFeedbackRepository) *FeedbackHandler {
	return &FeedbackHandler{feedbackService: feedbackService, feedbackRepo: feedbackRepo}
}

// parseFeedbackPath checks that the authenticated user matches {user_id} and
// parses both IDs. When it fails it has already written the error response.
func parseFeedbackPath(w http.ResponseWriter, r *http.Request) (userID, contentID uuid.UUID, ok bool) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Authentication context error", nil, "v1")
		return uuid.Nil, uuid.Nil, false
	}

	userID, err = uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return uuid.Nil, uuid.Nil, false
	}

	if authenticatedUserID != userID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own content", nil, "v1")
		return uuid.Nil, uuid.Nil, false
	}

	contentID, err = uuid.Parse(chi.URLParam(r, "content_id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid content ID format", nil, "v1")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, contentID, true
}

// PutFeedback handles PUT /api/v1/content/user/:user_id/:content_id/feedback
func (h *FeedbackHandler) PutFeedback(w http.ResponseWriter, r *http.Request) {
	userID, contentID, ok := parseFeedbackPath(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSimpleRequestSize)

	var req dto.ExtractionFeedbackRequest
	if err := middleware.DecodeJSONBody(r, &req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	if err := req.Validate(); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, err.Error(), nil, "v1")
		return
	}

	feedback, err := h.feedbackService.Submit(r.Context(), userID, contentID, req.Categories, req.Comment)
	if errors.Is(err, service.ErrUserContentNotFound) {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "User content not found", nil, "v1")
		return
	}
	if err != nil {
		slog.Error("failed to save extraction feedback", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to save feedback", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, dto.NewExtractionFeedbackResponse(feedback), "v1")
}

// GetFeedback handles GET /api/v1/content/user/:user_id/:content_id/feedback
func (h *FeedbackHandler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	userID, contentID, ok := parseFeedbackPath(w, r)
	if !ok {
		return
	}

	feedback, err := h.feedbackRepo.GetByUserAndContent(r.Context(), userID, contentID)
	if err != nil {
		slog.Error("failed to fetch extraction feedback", slog.Any("error", err))
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to fetch feedback", nil, "v1")
		return
	}
	if feedback == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "Feedback not found", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusOK, dto.NewExtractionFeedbackResponse(feedback), "v1")
}
