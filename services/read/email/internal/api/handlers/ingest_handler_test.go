package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/service"
	"github.com/stretchr/testify/assert"
)

// mockEmailService implements service.EmailService for testing.
type mockEmailService struct {
	ingestFn func(ctx context.Context, input service.IngestEmailInput) (bool, error)
}

func (m *mockEmailService) IngestEmail(ctx context.Context, input service.IngestEmailInput) (bool, error) {
	return m.ingestFn(ctx, input)
}

func TestIngestEmail_Success_Accepted(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, input service.IngestEmailInput) (bool, error) {
			assert.Equal(t, "user123@cairn.email", input.Recipient)
			assert.Equal(t, "newsletter@example.com", input.SenderEmail)
			assert.Equal(t, "Example Newsletter", input.SenderName)
			assert.Equal(t, "Weekly Update", input.Subject)
			assert.Equal(t, "<p>Hello</p>", input.HTMLBody)
			assert.Equal(t, "Hello", input.TextBody)
			return true, nil
		},
	}

	handler := NewIngestHandler(mock)

	body := map[string]string{
		"recipient":   "user123@cairn.email",
		"sender":      "newsletter@example.com",
		"sender_name": "Example Newsletter",
		"subject":     "Weekly Update",
		"html_body":   "<p>Hello</p>",
		"text_body":   "Hello",
		"received_at": "2025-01-15T10:30:00Z",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp api.SuccessResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)

	dataMap, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, dataMap["accepted"])
}

func TestIngestEmail_Success_Rejected(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, _ service.IngestEmailInput) (bool, error) {
			return false, nil
		},
	}

	handler := NewIngestHandler(mock)

	body := map[string]string{
		"recipient": "unknown@cairn.email",
		"sender":    "newsletter@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp api.SuccessResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)

	dataMap, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, false, dataMap["accepted"])
}

func TestIngestEmail_InvalidJSON(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, _ service.IngestEmailInput) (bool, error) {
			t.Fatal("should not be called")
			return false, nil
		},
	}

	handler := NewIngestHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeBadRequest, resp.Error)
}

func TestIngestEmail_MissingRecipient(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, _ service.IngestEmailInput) (bool, error) {
			t.Fatal("should not be called")
			return false, nil
		},
	}

	handler := NewIngestHandler(mock)

	body := map[string]string{
		"sender": "newsletter@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeValidation, resp.Error)
	assert.Contains(t, resp.Message, "recipient and sender are required")
}

func TestIngestEmail_MissingSender(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, _ service.IngestEmailInput) (bool, error) {
			t.Fatal("should not be called")
			return false, nil
		},
	}

	handler := NewIngestHandler(mock)

	body := map[string]string{
		"recipient": "user123@cairn.email",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeValidation, resp.Error)
}

func TestIngestEmail_ServiceError(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, _ service.IngestEmailInput) (bool, error) {
			return false, errors.New("database unavailable")
		},
	}

	handler := NewIngestHandler(mock)

	body := map[string]string{
		"recipient": "user123@cairn.email",
		"sender":    "newsletter@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeInternal, resp.Error)
}

func TestIngestEmail_BodyTooLarge(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, _ service.IngestEmailInput) (bool, error) {
			t.Fatal("should not be called")
			return false, nil
		},
	}

	handler := NewIngestHandler(mock)

	// Pad well past the 10MB cap with a large HTML body.
	body := map[string]string{
		"recipient": "user123@cairn.email",
		"sender":    "newsletter@example.com",
		"html_body": strings.Repeat("a", 11<<20),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)

	var resp api.ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, api.ErrCodeBadRequest, resp.Error)
}

func TestIngestEmail_BodyUnderLimitSucceeds(t *testing.T) {
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, _ service.IngestEmailInput) (bool, error) {
			return true, nil
		},
	}

	handler := NewIngestHandler(mock)

	body := map[string]string{
		"recipient": "user123@cairn.email",
		"sender":    "newsletter@example.com",
		"html_body": strings.Repeat("a", 1<<20), // 1MB, well under the 10MB cap
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.IngestEmail(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestIngestEmail_InvalidReceivedAtFallsBackToNow(t *testing.T) {
	var capturedInput service.IngestEmailInput
	mock := &mockEmailService{
		ingestFn: func(_ context.Context, input service.IngestEmailInput) (bool, error) {
			capturedInput = input
			return true, nil
		},
	}

	handler := NewIngestHandler(mock)

	body := map[string]string{
		"recipient":   "user123@cairn.email",
		"sender":      "newsletter@example.com",
		"received_at": "not-a-date",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/ingest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	before := time.Now().UTC().Add(-time.Second)
	handler.IngestEmail(rec, req)
	after := time.Now().UTC().Add(time.Second)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	// The fallback time should be approximately now
	assert.True(t, capturedInput.ReceivedAt.After(before), "received_at should be after test start")
	assert.True(t, capturedInput.ReceivedAt.Before(after), "received_at should be before test end")
}
