package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogging_Success(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := Logging(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/contents", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	// Verify log output
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "[GET]")
	assert.Contains(t, logOutput, "/api/v1/contents")
	assert.Contains(t, logOutput, "Status: 200")
	assert.Contains(t, logOutput, "Duration:")
	assert.Contains(t, logOutput, "192.168.1.1:12345")
}

func TestLogging_WithDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			log.SetOutput(&logBuffer)
			defer log.SetOutput(io.Discard)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			middleware := Logging(nextHandler)

			req := httptest.NewRequest("POST", "/api/v1/test", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			// Verify the logged status code matches
			logOutput := logBuffer.String()
			assert.Contains(t, logOutput, "Status: ")
			// Check that the status code number is present
			assert.Contains(t, logOutput, fmt.Sprintf("%d", tt.statusCode))
		})
	}
}

func TestLogging_DifferentHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var logBuffer bytes.Buffer
			log.SetOutput(&logBuffer)
			defer log.SetOutput(io.Discard)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := Logging(nextHandler)

			req := httptest.NewRequest(method, "/api/v1/test", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			logOutput := logBuffer.String()
			assert.Contains(t, logOutput, "["+method+"]")
		})
	}
}

func TestLogging_CapturesDuration(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		w.WriteHeader(http.StatusOK)
	})

	middleware := Logging(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Duration:")
	// Duration should contain time units (ns, µs, ms, or s)
	assert.True(t,
		strings.Contains(logOutput, "ns") ||
			strings.Contains(logOutput, "µs") ||
			strings.Contains(logOutput, "ms") ||
			strings.Contains(logOutput, "s"),
		"log should contain duration with time units")
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestResponseWriter_WriteHeaderMultipleTimes(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// First write should work
	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)

	// Second write should be ignored
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusCreated, rw.statusCode, "status should not change after first write")
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	data := []byte("test data")
	n, err := rw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "test data", w.Body.String())
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseWriter_WriteBeforeWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Write without calling WriteHeader first
	// Should default to 200 OK
	data := []byte("test")
	rw.Write(data)

	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, rw.written)
}
