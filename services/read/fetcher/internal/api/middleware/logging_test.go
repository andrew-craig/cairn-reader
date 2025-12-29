package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger_Success(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := Logger(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/feeds?user_id=123", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	// Verify log output
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/api/v1/feeds?user_id=123")
	assert.Contains(t, logOutput, "Status: 200")
	assert.Contains(t, logOutput, "Duration:")
	assert.Contains(t, logOutput, "192.168.1.1:54321")
}

func TestLogger_WithDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"204 No Content", http.StatusNoContent},
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
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

			middleware := Logger(nextHandler)

			req := httptest.NewRequest("POST", "/api/v1/test", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			// Verify the logged status code matches
			logOutput := logBuffer.String()
			assert.Contains(t, logOutput, "Status: ")
			assert.Contains(t, logOutput, strings.TrimSpace(strings.Split(tt.name, " ")[0]))
		})
	}
}

func TestLogger_DifferentHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var logBuffer bytes.Buffer
			log.SetOutput(&logBuffer)
			defer log.SetOutput(io.Discard)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := Logger(nextHandler)

			req := httptest.NewRequest(method, "/api/v1/test", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			logOutput := logBuffer.String()
			assert.Contains(t, logOutput, method)
		})
	}
}

func TestLogger_CapturesDuration(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		w.WriteHeader(http.StatusOK)
	})

	middleware := Logger(nextHandler)

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

func TestLogger_LogsRequestURI(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Logger(nextHandler)

	// Test with query parameters
	req := httptest.NewRequest("GET", "/api/v1/feeds?limit=20&offset=10&status=active", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "/api/v1/feeds?limit=20&offset=10&status=active")
}

func TestNewResponseWriter_Creation(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	assert.NotNil(t, rw)
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.False(t, rw.written)
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	rw.WriteHeader(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestResponseWriter_WriteHeaderMultipleTimes(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	// First write should work
	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)

	// Second write should be ignored
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusCreated, rw.statusCode, "status should not change after first write")
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	data := []byte("test response data")
	n, err := rw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "test response data", w.Body.String())
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseWriter_WriteBeforeWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	// Write without calling WriteHeader first
	// Should default to 200 OK
	data := []byte("test")
	rw.Write(data)

	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, rw.written)
}

func TestResponseWriter_MultipleWrites(t *testing.T) {
	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("part1 "))
	rw.Write([]byte("part2 "))
	rw.Write([]byte("part3"))

	assert.Equal(t, "part1 part2 part3", w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_WithEmptyResponse(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	middleware := Logger(nextHandler)

	req := httptest.NewRequest("DELETE", "/api/v1/feeds/123", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "DELETE")
	assert.Contains(t, logOutput, "Status: 204")
}

func TestLogger_WithLargeResponse(t *testing.T) {
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(io.Discard)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write a large response
		largeData := strings.Repeat("a", 10000)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeData))
	})

	middleware := Logger(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 10000, w.Body.Len())

	// Logging should still work
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "Status: 200")
}
