package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewMiddleware(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)

	middleware := NewMiddleware(validator)

	if middleware == nil {
		t.Fatal("Expected middleware to be created, got nil")
	}

	if middleware.validator != validator {
		t.Error("Validator not set correctly in middleware")
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	// Create a handler that checks if user ID is in context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextUserID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Error("Expected user ID in context")
		}
		if contextUserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, contextUserID)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	protectedHandler := middleware.RequireAuth(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequireAuth_MissingToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with missing token")
	})

	protectedHandler := middleware.RequireAuth(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid token")
	})

	protectedHandler := middleware.RequireAuth(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)
	userID := uuid.New()

	// Create expired token
	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", -1*time.Hour)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with expired token")
	})

	protectedHandler := middleware.RequireAuth(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireAuth_InvalidAuthHeader(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid auth header")
	})

	protectedHandler := middleware.RequireAuth(handler)

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "token123"},
		{"wrong scheme", "Basic token123"},
		{"empty bearer", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			protectedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", w.Code)
			}
		})
	}
}

func TestOptionalAuth_ValidToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r.Context()) {
			t.Error("Expected request to be authenticated")
		}

		contextUserID := MustGetUserID(r.Context())
		if contextUserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, contextUserID)
		}

		w.WriteHeader(http.StatusOK)
	})

	optionalHandler := middleware.OptionalAuth(handler)

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	optionalHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOptionalAuth_MissingToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		if IsAuthenticated(r.Context()) {
			t.Error("Expected request to not be authenticated")
		}
		w.WriteHeader(http.StatusOK)
	})

	optionalHandler := middleware.OptionalAuth(handler)

	req := httptest.NewRequest("GET", "/optional", nil)
	w := httptest.NewRecorder()

	optionalHandler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called even without token")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOptionalAuth_InvalidToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		if IsAuthenticated(r.Context()) {
			t.Error("Expected request to not be authenticated with invalid token")
		}
		w.WriteHeader(http.StatusOK)
	})

	optionalHandler := middleware.OptionalAuth(handler)

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")
	w := httptest.NewRecorder()

	optionalHandler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called even with invalid token")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	userID := uuid.New()

	t.Run("with user ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDContextKey, userID)

		retrievedID, ok := GetUserIDFromContext(ctx)
		if !ok {
			t.Error("Expected to find user ID in context")
		}
		if retrievedID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, retrievedID)
		}
	})

	t.Run("without user ID", func(t *testing.T) {
		ctx := context.Background()

		_, ok := GetUserIDFromContext(ctx)
		if ok {
			t.Error("Expected not to find user ID in context")
		}
	})

	t.Run("with invalid type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDContextKey, "not-a-uuid")

		_, ok := GetUserIDFromContext(ctx)
		if ok {
			t.Error("Expected not to find user ID with invalid type")
		}
	})
}

func TestMustGetUserID(t *testing.T) {
	userID := uuid.New()

	t.Run("with user ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDContextKey, userID)

		retrievedID := MustGetUserID(ctx)
		if retrievedID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, retrievedID)
		}
	})

	t.Run("without user ID panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected MustGetUserID to panic")
			}
		}()

		ctx := context.Background()
		MustGetUserID(ctx)
	})
}

func TestIsAuthenticated(t *testing.T) {
	userID := uuid.New()

	t.Run("authenticated", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDContextKey, userID)

		if !IsAuthenticated(ctx) {
			t.Error("Expected context to be authenticated")
		}
	})

	t.Run("not authenticated", func(t *testing.T) {
		ctx := context.Background()

		if IsAuthenticated(ctx) {
			t.Error("Expected context to not be authenticated")
		}
	})
}

func TestChain(t *testing.T) {
	// Define custom context key types to avoid collisions
	type testContextKey string
	const (
		testKey1 testContextKey = "key1"
		testKey2 testContextKey = "key2"
	)

	// Create middleware that adds values to context
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), testKey1, "value1")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), testKey2, "value2")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(testKey1) != "value1" {
			t.Error("Expected key1 to be in context")
		}
		if r.Context().Value(testKey2) != "value2" {
			t.Error("Expected key2 to be in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Chain middleware
	chained := Chain(middleware1, middleware2)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMiddleware_ErrorResponse(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	protectedHandler := middleware.RequireAuth(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	// Check Content-Type
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type to be application/json")
	}

	// Check status code
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Response body should contain error JSON
	body := w.Body.String()
	if body == "" {
		t.Error("Expected error response body")
	}
}

func BenchmarkRequireAuth_ValidToken(b *testing.B) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}
	publicKey := &privateKey.PublicKey

	validator := NewValidator(publicKey)
	middleware := NewMiddleware(validator)
	userID := uuid.New()

	token := generateTestToken(&testing.T{}, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := middleware.RequireAuth(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)
	}
}
