package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/api/dto"
	"github.com/stretchr/testify/assert"
)

func TestWriteError_BasicError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "validation_error", "Invalid input", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response dto.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
	assert.Equal(t, "Invalid input", response.Message)
	assert.Nil(t, response.Details)
}

func TestWriteError_WithDetails(t *testing.T) {
	w := httptest.NewRecorder()

	details := map[string]string{
		"field1": "error1",
		"field2": "error2",
	}

	WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Validation failed", details)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response dto.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_failed", response.Error)
	assert.Equal(t, "Validation failed", response.Message)
	assert.NotNil(t, response.Details)
	assert.Equal(t, "error1", response.Details["field1"])
	assert.Equal(t, "error2", response.Details["field2"])
}

func TestWriteError_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorType  string
		message    string
	}{
		{"Bad Request", http.StatusBadRequest, "bad_request", "Bad request"},
		{"Unauthorized", http.StatusUnauthorized, "unauthorized", "Unauthorized"},
		{"Forbidden", http.StatusForbidden, "forbidden", "Forbidden"},
		{"Not Found", http.StatusNotFound, "not_found", "Not found"},
		{"Conflict", http.StatusConflict, "conflict", "Conflict"},
		{"Internal Server Error", http.StatusInternalServerError, "internal_error", "Internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteError(w, tt.statusCode, tt.errorType, tt.message, nil)

			assert.Equal(t, tt.statusCode, w.Code)

			var response dto.ErrorResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Equal(t, tt.errorType, response.Error)
			assert.Equal(t, tt.message, response.Message)
		})
	}
}

func TestWriteJSON_SuccessResponse(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"id":    "123",
		"title": "Test Article",
		"count": 42,
	}

	WriteJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "123", response["id"])
	assert.Equal(t, "Test Article", response["title"])
	assert.Equal(t, float64(42), response["count"]) // JSON numbers decode as float64
}

func TestWriteJSON_StructResponse(t *testing.T) {
	w := httptest.NewRecorder()

	type TestResponse struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	data := TestResponse{
		ID:    "456",
		Name:  "Test",
		Count: 10,
	}

	WriteJSON(w, http.StatusCreated, data)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response TestResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "456", response.ID)
	assert.Equal(t, "Test", response.Name)
	assert.Equal(t, 10, response.Count)
}

func TestWriteJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusNoContent, nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestWriteJSON_EmptySlice(t *testing.T) {
	w := httptest.NewRecorder()

	data := []string{}
	WriteJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Empty(t, response)
}
