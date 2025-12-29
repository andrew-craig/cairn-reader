package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateJSON_ValidContentType(t *testing.T) {
	// Create a test handler that should be called
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with ValidateJSON middleware
	middleware := ValidateJSON(nextHandler)

	// Create a POST request with valid JSON content type
	req := httptest.NewRequest("POST", "/api/v1/contents", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute the middleware
	middleware.ServeHTTP(w, req)

	// Assertions
	assert.True(t, called, "next handler should be called")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateJSON_ValidContentTypeWithCharset(t *testing.T) {
	// Create a test handler that should be called
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := ValidateJSON(nextHandler)

	// Create a POST request with JSON content type including charset
	req := httptest.NewRequest("POST", "/api/v1/contents", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.True(t, called, "next handler should be called")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateJSON_InvalidContentType(t *testing.T) {
	// Create a test handler that should NOT be called
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	middleware := ValidateJSON(nextHandler)

	// Create a POST request with invalid content type
	req := httptest.NewRequest("POST", "/api/v1/contents", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_content_type")
	assert.Contains(t, w.Body.String(), "Content-Type must be application/json")
}

func TestValidateJSON_MissingContentType(t *testing.T) {
	// Create a test handler that should NOT be called
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	middleware := ValidateJSON(nextHandler)

	// Create a POST request without Content-Type header
	req := httptest.NewRequest("POST", "/api/v1/contents", strings.NewReader(`{"test":"data"}`))
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_content_type")
}

func TestValidateJSON_PUTRequest(t *testing.T) {
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := ValidateJSON(nextHandler)

	// Create a PUT request with valid JSON content type
	req := httptest.NewRequest("PUT", "/api/v1/contents/123", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.True(t, called, "next handler should be called for PUT")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateJSON_PATCHRequest(t *testing.T) {
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := ValidateJSON(nextHandler)

	// Create a PATCH request with valid JSON content type
	req := httptest.NewRequest("PATCH", "/api/v1/contents/123", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.True(t, called, "next handler should be called for PATCH")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateJSON_GETRequest(t *testing.T) {
	// GET requests should bypass content-type validation
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := ValidateJSON(nextHandler)

	// Create a GET request without Content-Type header
	req := httptest.NewRequest("GET", "/api/v1/contents", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.True(t, called, "next handler should be called for GET")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateJSON_DELETERequest(t *testing.T) {
	// DELETE requests should bypass content-type validation
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := ValidateJSON(nextHandler)

	// Create a DELETE request without Content-Type header
	req := httptest.NewRequest("DELETE", "/api/v1/contents/123", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.True(t, called, "next handler should be called for DELETE")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDecodeJSONBody_ValidJSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	jsonBody := `{"name":"test","value":123}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(jsonBody))

	var result TestStruct
	err := DecodeJSONBody(req, &result)

	assert.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 123, result.Value)
}

func TestDecodeJSONBody_InvalidJSON(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	jsonBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(jsonBody))

	var result TestStruct
	err := DecodeJSONBody(req, &result)

	assert.Error(t, err)
}

func TestDecodeJSONBody_EmptyBody(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest("POST", "/test", strings.NewReader(""))

	var result TestStruct
	err := DecodeJSONBody(req, &result)

	assert.Equal(t, io.EOF, err)
}

func TestDecodeJSONBody_UnknownFields(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	// JSON with unknown field "extra"
	jsonBody := `{"name":"test","extra":"unknown"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(jsonBody))

	var result TestStruct
	err := DecodeJSONBody(req, &result)

	// Should fail because DisallowUnknownFields is set
	assert.Error(t, err)
}

func TestValidateStatus_ValidStatuses(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"unread", true},
		{"read", true},
		{"archived", true},
		{"reading", false},   // Not a valid status
		{"completed", false}, // Not a valid status
		{"invalid", false},
		{"", false},
		{"UNREAD", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := ValidateStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSourceType_ValidTypes(t *testing.T) {
	tests := []struct {
		sourceType string
		expected   bool
	}{
		{"rss", true},
		{"web", true},
		{"invalid", false},
		{"", false},
		{"RSS", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.sourceType, func(t *testing.T) {
			result := ValidateSourceType(tt.sourceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
