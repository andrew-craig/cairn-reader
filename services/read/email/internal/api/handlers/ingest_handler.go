package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/service"
)

// maxIngestEmailBodySize bounds the ingest request body (email HTML + text
// bodies) to prevent unbounded memory use from an oversized POST.
const maxIngestEmailBodySize = 10 << 20 // 10MB

// IngestHandler handles email ingestion from Cloudflare Email Worker.
type IngestHandler struct {
	emailService service.EmailService
}

// NewIngestHandler creates a new IngestHandler.
func NewIngestHandler(emailService service.EmailService) *IngestHandler {
	return &IngestHandler{emailService: emailService}
}

// IngestEmail handles POST /api/v1/source/email/ingest
func (h *IngestHandler) IngestEmail(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxIngestEmailBodySize)

	var req dto.IngestEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, api.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	if req.Recipient == "" || req.Sender == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeValidation, "recipient and sender are required", nil, "v1")
		return
	}

	receivedAt, err := time.Parse(time.RFC3339, req.ReceivedAt)
	if err != nil {
		receivedAt = time.Now().UTC()
	}

	accepted, err := h.emailService.IngestEmail(r.Context(), service.IngestEmailInput{
		Recipient:   req.Recipient,
		SenderEmail: req.Sender,
		SenderName:  req.SenderName,
		Subject:     req.Subject,
		HTMLBody:    req.HTMLBody,
		TextBody:    req.TextBody,
		ReceivedAt:  receivedAt,
	})
	if err != nil {
		slog.Error("failed to ingest email", "error", err)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to ingest email", nil, "v1")
		return
	}

	api.WriteSuccess(w, http.StatusAccepted, dto.IngestEmailResponse{Accepted: accepted}, "v1")
}
