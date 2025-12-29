package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecovery_NoPanic(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := Recovery(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should complete successfully
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	// Should not log any panic
	logOutput := logBuffer.String()
	assert.NotContains(t, logOutput, "PANIC")
}

func TestRecovery_WithPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := Recovery(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should return 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Should return JSON error response
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response["error"])
	assert.Equal(t, "An internal server error occurred", response["message"])
	assert.Equal(t, float64(http.StatusInternalServerError), response["code"])

	// Should log the panic with stack trace
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "PANIC")
	assert.Contains(t, logOutput, "test panic")
	// Stack trace should be included
	assert.Contains(t, logOutput, "goroutine")
}

func TestRecovery_WithStringPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("string panic message")
	})

	middleware := Recovery(nextHandler)

	req := httptest.NewRequest("POST", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response["error"])

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "string panic message")
}

func TestRecovery_WithErrorPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(io.EOF)
	})

	middleware := Recovery(nextHandler)

	req := httptest.NewRequest("PUT", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response["error"])

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "PANIC")
}

func TestRecovery_WithNilPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ptr *string
		_ = *ptr // This will cause a nil pointer panic
	})

	middleware := Recovery(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response["error"])
}

func TestRecovery_StackTraceIncluded(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test for stack trace")
	})

	middleware := Recovery(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	logOutput := logBuffer.String()

	// Stack trace should include:
	// - The panic message
	assert.Contains(t, logOutput, "test for stack trace")
	// - Goroutine information
	assert.Contains(t, logOutput, "goroutine")
	// - File and line information (from debug.Stack())
	// This is a basic check that debug.Stack() was called
	assert.True(t, len(logOutput) > 100, "stack trace should be substantial")
}

func TestRecovery_MultiplePanicsDifferentHandlers(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	// First handler panics
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic 1")
	})

	// Second handler panics differently
	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic 2")
	})

	middleware1 := Recovery(handler1)
	middleware2 := Recovery(handler2)

	// Test first panic
	req1 := httptest.NewRequest("GET", "/api/v1/test1", nil)
	w1 := httptest.NewRecorder()
	middleware1.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusInternalServerError, w1.Code)

	// Test second panic
	req2 := httptest.NewRequest("GET", "/api/v1/test2", nil)
	w2 := httptest.NewRecorder()
	middleware2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusInternalServerError, w2.Code)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "panic 1")
	assert.Contains(t, logOutput, "panic 2")
}

func TestRecovery_JSONResponseFormat(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	middleware := Recovery(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify JSON structure
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	// Check all required fields are present
	assert.Contains(t, response, "error")
	assert.Contains(t, response, "message")
	assert.Contains(t, response, "code")

	// Check field types and values
	assert.IsType(t, "", response["error"])
	assert.IsType(t, "", response["message"])
	assert.IsType(t, float64(0), response["code"])
}
