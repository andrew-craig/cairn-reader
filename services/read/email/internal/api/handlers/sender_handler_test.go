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
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockSenderService implements service.SenderService for testing.
type mockSenderService struct {
	upsertOnReceiptFn func(ctx context.Context, userID uuid.UUID, senderEmail string, senderName string, receivedAt time.Time) (*models.EmailSender, error)
	listByUserFn      func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error)
}

func (m *mockSenderService) UpsertOnReceipt(ctx context.Context, userID uuid.UUID, senderEmail string, senderName string, receivedAt time.Time) (*models.EmailSender, error) {
	return m.upsertOnReceiptFn(ctx, userID, senderEmail, senderName, receivedAt)
}

func (m *mockSenderService) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
	return m.listByUserFn(ctx, userID, limit, offset)
}

func setupSenderRequest(method, url string, userIDParam string, authedUserID *uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, url, nil)

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("user_id", userIDParam)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx)

	if authedUserID != nil {
		ctx = auth.SetUserIDInContext(ctx, *authedUserID)
	}

	return req.WithContext(ctx)
}

func TestListSenders_Success(t *testing.T) {
	userID := uuid.New()
	senderID1 := uuid.New()
	senderID2 := uuid.New()
	senderName := "Newsletter Co"
	lastReceived := time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)

	mock := &mockSenderService{
		listByUserFn: func(_ context.Context, id uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
			assert.Equal(t, userID, id)
			assert.Equal(t, 20, limit) // default
			assert.Equal(t, 0, offset) // default
			return []*models.EmailSender{
				{
					ID:             senderID1,
					UserID:         userID,
					SenderEmail:    "news@example.com",
					SenderName:     &senderName,
					EmailCount:     42,
					LastReceivedAt: &lastReceived,
				},
				{
					ID:          senderID2,
					UserID:      userID,
					SenderEmail: "alerts@example.com",
					SenderName:  nil,
					EmailCount:  5,
				},
			}, nil
		},
	}

	handler := NewSenderHandler(mock)
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/senders", userID.String(), &userID)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.SuccessResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)

	dataSlice, ok := resp.Data.([]interface{})
	assert.True(t, ok)
	assert.Len(t, dataSlice, 2)

	first, ok := dataSlice[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, senderID1.String(), first["id"])
	assert.Equal(t, "news@example.com", first["sender_email"])
	assert.Equal(t, "Newsletter Co", first["sender_name"])
	assert.Equal(t, float64(42), first["email_count"])
	assert.Equal(t, "2025-06-10T12:00:00Z", first["last_received_at"])

	second, ok := dataSlice[1].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, senderID2.String(), second["id"])
	assert.Equal(t, "alerts@example.com", second["sender_email"])
	// sender_name omitted when nil
	_, hasSenderName := second["sender_name"]
	assert.False(t, hasSenderName)
	assert.Equal(t, float64(5), second["email_count"])
}

func TestListSenders_EmptyList(t *testing.T) {
	userID := uuid.New()

	mock := &mockSenderService{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.EmailSender, error) {
			return []*models.EmailSender{}, nil
		},
	}

	handler := NewSenderHandler(mock)
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/senders", userID.String(), &userID)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp api.SuccessResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)

	dataSlice, ok := resp.Data.([]interface{})
	assert.True(t, ok)
	assert.Len(t, dataSlice, 0)
}

func TestListSenders_NoJWTContext(t *testing.T) {
	mock := &mockSenderService{}
	handler := NewSenderHandler(mock)

	userID := uuid.New()
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/senders", userID.String(), nil)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeUnauthorized, resp.Error)
}

func TestListSenders_UserMismatch(t *testing.T) {
	mock := &mockSenderService{}
	handler := NewSenderHandler(mock)

	requestedUserID := uuid.New()
	authedUserID := uuid.New()
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+requestedUserID.String()+"/senders", requestedUserID.String(), &authedUserID)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeForbidden, resp.Error)
}

func TestListSenders_ServiceError(t *testing.T) {
	userID := uuid.New()

	mock := &mockSenderService{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.EmailSender, error) {
			return nil, errors.New("database connection lost")
		},
	}

	handler := NewSenderHandler(mock)
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/senders", userID.String(), &userID)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeInternal, resp.Error)
}

func TestListSenders_CustomLimitOffset(t *testing.T) {
	userID := uuid.New()

	mock := &mockSenderService{
		listByUserFn: func(_ context.Context, _ uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
			assert.Equal(t, 50, limit)
			assert.Equal(t, 10, offset)
			return []*models.EmailSender{}, nil
		},
	}

	handler := NewSenderHandler(mock)
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/senders?limit=50&offset=10", userID.String(), &userID)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListSenders_InvalidUserID(t *testing.T) {
	mock := &mockSenderService{}
	handler := NewSenderHandler(mock)

	authedUserID := uuid.New()
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/not-a-uuid/senders", "not-a-uuid", &authedUserID)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeBadRequest, resp.Error)
}

func TestListSenders_LimitDefaults(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedLimit int
	}{
		{"negative limit defaults to 20", "?limit=-5", 20},
		{"zero limit defaults to 20", "?limit=0", 20},
		{"over 100 defaults to 20", "?limit=200", 20},
		{"valid limit used", "?limit=75", 75},
		{"no limit defaults to 20", "", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()

			mock := &mockSenderService{
				listByUserFn: func(_ context.Context, _ uuid.UUID, limit, _ int) ([]*models.EmailSender, error) {
					assert.Equal(t, tt.expectedLimit, limit)
					return []*models.EmailSender{}, nil
				},
			}

			handler := NewSenderHandler(mock)
			req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/senders"+tt.query, userID.String(), &userID)
			rec := httptest.NewRecorder()

			handler.ListSenders(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestListSenders_NegativeOffsetDefaultsToZero(t *testing.T) {
	userID := uuid.New()

	mock := &mockSenderService{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _, offset int) ([]*models.EmailSender, error) {
			assert.Equal(t, 0, offset)
			return []*models.EmailSender{}, nil
		},
	}

	handler := NewSenderHandler(mock)
	req := setupSenderRequest(http.MethodGet, "/api/v1/email/user/"+userID.String()+"/senders?offset=-5", userID.String(), &userID)
	rec := httptest.NewRecorder()

	handler.ListSenders(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
