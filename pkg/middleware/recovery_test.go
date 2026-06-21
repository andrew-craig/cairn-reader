package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovery(t *testing.T) {
	t.Run("passes through without panic", func(t *testing.T) {
		called := false
		handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !called {
			t.Error("handler should have been called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("recovers from panic and returns 500", func(t *testing.T) {
		handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if body["error"] == "" {
			t.Error("expected error field in response")
		}
		if body["message"] == "" {
			t.Error("expected message field in response")
		}
		details, ok := body["details"].(map[string]interface{})
		if !ok || details["request_id"] == "" {
			t.Error("expected request_id in details")
		}
		meta, ok := body["meta"].(map[string]interface{})
		if !ok || meta["version"] == "" {
			t.Error("expected meta.version in response")
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("does not expose internal details", func(t *testing.T) {
		handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("secret internal error details")
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		body := w.Body.String()
		if contains(body, "secret internal error details") {
			t.Error("response should not contain panic message")
		}
		if contains(body, "stack") {
			t.Error("response should not contain stack trace")
		}
	})
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For with single IP",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Forwarded-For with multiple IPs",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "10.0.0.1"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "falls back to RemoteAddr",
			headers:  map[string]string{},
			remote:   "192.168.1.1:1234",
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remote

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
