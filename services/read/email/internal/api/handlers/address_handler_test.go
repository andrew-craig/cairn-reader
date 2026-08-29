package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/config"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockAddressService implements service.AddressService for testing.
type mockAddressService struct {
	getOrCreateFn      func(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error)
	getByUserIDFn      func(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error)
	resolveRecipientFn func(ctx context.Context, recipient string) (uuid.UUID, error)
}

func (m *mockAddressService) GetOrCreate(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error) {
	return m.getOrCreateFn(ctx, userID)
}

func (m *mockAddressService) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error) {
	return m.getByUserIDFn(ctx, userID)
}

func (m *mockAddressService) ResolveRecipient(ctx context.Context, recipient string) (uuid.UUID, error) {
	return m.resolveRecipientFn(ctx, recipient)
}

func newAddressHandler(mock *mockAddressService) *AddressHandler {
	return NewAddressHandler(mock, config.EmailConfig{Domain: "cairn.email"})
}

func setupAddressRequest(method, url string, userID uuid.UUID, authedUserID *uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, url, nil)

	// Set chi URL params
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("user_id", userID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx)

	// Set authenticated user in context
	if authedUserID != nil {
		ctx = auth.SetUserIDInContext(ctx, *authedUserID)
	}

	return req.WithContext(ctx)
}

// --- GetOrCreate Tests ---

func TestGetOrCreate_Success(t *testing.T) {
	userID := uuid.New()
	createdAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	mock := &mockAddressService{
		getOrCreateFn: func(_ context.Context, id uuid.UUID) (*models.EmailAddress, error) {
			assert.Equal(t, userID, id)
			return &models.EmailAddress{
				ID:        uuid.New(),
				UserID:    userID,
				LocalPart: "k7m2x9pq",
				CreatedAt: createdAt,
			}, nil
		},
	}

	handler := newAddressHandler(mock)
	req := setupAddressRequest(http.MethodPost, "/api/v1/email/user/"+userID.String()+"/address", userID, &userID)
	rec := httptest.NewRecorder()

	handler.GetOrCreate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.SuccessResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)

	dataMap, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "k7m2x9pq@cairn.email", dataMap["email_address"])
	assert.Equal(t, "2025-01-15T10:00:00Z", dataMap["created_at"])
}

func TestGetOrCreate_NoJWTContext(t *testing.T) {
	mock := &mockAddressService{}
	handler := newAddressHandler(mock)

	userID := uuid.New()
	// No authenticated user in context
	req := setupAddressRequest(http.MethodPost, "/api/v1/email/user/"+userID.String()+"/address", userID, nil)
	rec := httptest.NewRecorder()

	handler.GetOrCreate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeUnauthorized, resp.Error)
}

func TestGetOrCreate_UserMismatch(t *testing.T) {
	mock := &mockAddressService{}
	handler := newAddressHandler(mock)

	requestedUserID := uuid.New()
	authedUserID := uuid.New() // different user
	req := setupAddressRequest(http.MethodPost, "/api/v1/email/user/"+requestedUserID.String()+"/address", requestedUserID, &authedUserID)
	rec := httptest.NewRecorder()

	handler.GetOrCreate(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeForbidden, resp.Error)
}

func TestGetOrCreate_InvalidUserID(t *testing.T) {
	mock := &mockAddressService{}
	handler := newAddressHandler(mock)

	authedUserID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/user/not-a-uuid/address", nil)
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("user_id", "not-a-uuid")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx)
	ctx = auth.SetUserIDInContext(ctx, authedUserID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.GetOrCreate(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeBadRequest, resp.Error)
}

func TestGetOrCreate_ServiceError(t *testing.T) {
	userID := uuid.New()

	mock := &mockAddressService{
		getOrCreateFn: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, errors.New("database error")
		},
	}

	handler := newAddressHandler(mock)
	req := setupAddressRequest(http.MethodPost, "/api/v1/email/user/"+userID.String()+"/address", userID, &userID)
	rec := httptest.NewRecorder()

	handler.GetOrCreate(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeInternal, resp.Error)
}

// --- Get Tests ---

func TestGet_Success(t *testing.T) {
	userID := uuid.New()
	createdAt := time.Date(2025, 3, 20, 14, 30, 0, 0, time.UTC)

	mock := &mockAddressService{
		getByUserIDFn: func(_ context.Context, id uuid.UUID) (*models.EmailAddress, error) {
			assert.Equal(t, userID, id)
			return &models.EmailAddress{
				ID:        uuid.New(),
				UserID:    userID,
				LocalPart: "abc12345",
				CreatedAt: createdAt,
			}, nil
		},
	}

	handler := newAddressHandler(mock)
	req := setupAddressRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/address", userID, &userID)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.SuccessResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)

	dataMap, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "abc12345@cairn.email", dataMap["email_address"])
	assert.Equal(t, "2025-03-20T14:30:00Z", dataMap["created_at"])
}

func TestGet_NotFound(t *testing.T) {
	userID := uuid.New()

	mock := &mockAddressService{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, nil
		},
	}

	handler := newAddressHandler(mock)
	req := setupAddressRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/address", userID, &userID)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeNotFound, resp.Error)
}

func TestGet_NoJWTContext(t *testing.T) {
	mock := &mockAddressService{}
	handler := newAddressHandler(mock)

	userID := uuid.New()
	req := setupAddressRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/address", userID, nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeUnauthorized, resp.Error)
}
