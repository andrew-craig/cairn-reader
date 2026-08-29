package handlers

import (
	"log/slog"
	"net/http"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/api/dto"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/config"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AddressHandler handles email address management endpoints.
type AddressHandler struct {
	addressService service.AddressService
	emailDomain    string
}

// NewAddressHandler creates a new AddressHandler.
func NewAddressHandler(addressService service.AddressService, emailCfg config.EmailConfig) *AddressHandler {
	return &AddressHandler{
		addressService: addressService,
		emailDomain:    emailCfg.Domain,
	}
}

// GetOrCreate handles POST /api/v1/source/email/user/{user_id}/address
func (h *AddressHandler) GetOrCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorizeUser(w, r)
	if !ok {
		return
	}

	address, err := h.addressService.GetOrCreate(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get or create email address", "error", err, "user_id", userID)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to create email address", nil, "v1")
		return
	}

	resp := dto.CreateAddressResponse{}
	resp.Data.EmailAddress = address.LocalPart + "@" + h.emailDomain
	resp.Data.CreatedAt = address.CreatedAt.Format("2006-01-02T15:04:05Z")

	api.WriteSuccess(w, http.StatusOK, resp.Data, "v1")
}

// Get handles GET /api/v1/source/email/user/{user_id}/address
func (h *AddressHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorizeUser(w, r)
	if !ok {
		return
	}

	address, err := h.addressService.GetByUserID(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get email address", "error", err, "user_id", userID)
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to get email address", nil, "v1")
		return
	}

	if address == nil {
		api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "No email address found", nil, "v1")
		return
	}

	resp := dto.GetAddressResponse{}
	resp.Data.EmailAddress = address.LocalPart + "@" + h.emailDomain
	resp.Data.CreatedAt = address.CreatedAt.Format("2006-01-02T15:04:05Z")

	api.WriteSuccess(w, http.StatusOK, resp.Data, "v1")
}

// authorizeUser extracts the JWT user ID, parses the URL user_id param, and checks they match.
func (h *AddressHandler) authorizeUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "Missing or invalid authentication token", nil, "v1")
		return uuid.Nil, false
	}

	requestedUserID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "Invalid user ID format", nil, "v1")
		return uuid.Nil, false
	}

	if authenticatedUserID != requestedUserID {
		api.WriteError(w, http.StatusForbidden, api.ErrCodeForbidden, "User can only access their own resources", nil, "v1")
		return uuid.Nil, false
	}

	return requestedUserID, true
}
