package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SenderHandler handles sender listing endpoints.
type SenderHandler struct {
	senderService service.SenderService
}

// NewSenderHandler creates a new SenderHandler.
func NewSenderHandler(senderService service.SenderService) *SenderHandler {
	return &SenderHandler{senderService: senderService}
}

// ListSenders handles GET /api/v1/source/email/user/{user_id}/senders
func (h *SenderHandler) ListSenders(w http.ResponseWriter, r *http.Request) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "Missing or invalid authentication token", nil, "v1")
		return
	}

	requestedUserID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return
	}

	if authenticatedUserID != requestedUserID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own resources", nil, "v1")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	senders, err := h.senderService.ListByUser(r.Context(), requestedUserID, limit, offset)
	if err != nil {
		slog.Error("failed to list senders", "error", err, "user_id", requestedUserID)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to list senders", nil, "v1")
		return
	}

	resp := dto.ListSendersResponse{
		Data: make([]dto.SenderResponse, 0, len(senders)),
	}
	for _, s := range senders {
		sr := dto.SenderResponse{
			ID:          s.ID.String(),
			SenderEmail: s.SenderEmail,
			EmailCount:  s.EmailCount,
		}
		if s.SenderName != nil {
			sr.SenderName = *s.SenderName
		}
		if s.LastReceivedAt != nil {
			sr.LastReceivedAt = s.LastReceivedAt.Format("2006-01-02T15:04:05Z")
		}
		resp.Data = append(resp.Data, sr)
	}

	api.WriteSuccess(w, http.StatusOK, resp.Data, "v1")
}
