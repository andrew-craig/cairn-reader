package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// withChiURLParams adds Chi URL parameters to a request context
func withChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// TestRequireSameUser tests the RequireSameUser middleware
func TestRequireSameUser(t *testing.T) {
	t.Run("Authorized access (same user)", func(t *testing.T) {
		userID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handlerCalled := false
		middleware := RequireSameUser()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware(handler).ServeHTTP(w, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Unauthorized access (different user)", func(t *testing.T) {
		authenticatedUserID := uuid.New()
		requestedUserID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/users/"+requestedUserID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": requestedUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), authenticatedUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireSameUser()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "you can only access your own resources")
	})

	t.Run("Missing authentication (401)", func(t *testing.T) {
		requestedUserID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/users/"+requestedUserID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": requestedUserID.String()})
		// Don't set user ID in context

		w := httptest.NewRecorder()

		middleware := RequireSameUser()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authentication required")
	})

	t.Run("Missing ID parameter (400)", func(t *testing.T) {
		userID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)
		// No :id parameter

		w := httptest.NewRecorder()

		middleware := RequireSameUser()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "user ID parameter is required")
	})

	t.Run("Invalid UUID format (400)", func(t *testing.T) {
		userID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/users/invalid-uuid", nil)
		req = withChiURLParams(req, map[string]string{"id": "invalid-uuid"})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireSameUser()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user ID format")
	})

	t.Run("Request continues when authorized", func(t *testing.T) {
		userID := uuid.New()
		handlerCalled := false

		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireSameUser()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware(handler).ServeHTTP(w, req)

		assert.True(t, handlerCalled, "Handler should have been called")
	})
}

// TestRequireSameUserWithCustomParam tests the RequireSameUserWithCustomParam middleware
func TestRequireSameUserWithCustomParam(t *testing.T) {
	t.Run("Custom parameter name handling", func(t *testing.T) {
		userID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/profiles/"+userID.String(), nil)
		req = withChiURLParams(req, map[string]string{"user_id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handlerCalled := false
		middleware := RequireSameUserWithCustomParam("user_id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware(handler).ServeHTTP(w, req)

		assert.True(t, handlerCalled)
	})

	t.Run("Unauthorized with custom param", func(t *testing.T) {
		authenticatedUserID := uuid.New()
		requestedUserID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/profiles/"+requestedUserID.String(), nil)
		req = withChiURLParams(req, map[string]string{"user_id": requestedUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), authenticatedUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireSameUserWithCustomParam("user_id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Missing custom parameter", func(t *testing.T) {
		userID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/profiles", nil)
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireSameUserWithCustomParam("user_id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid UUID in custom parameter", func(t *testing.T) {
		userID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/profiles/not-a-uuid", nil)
		req = withChiURLParams(req, map[string]string{"user_id": "not-a-uuid"})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireSameUserWithCustomParam("user_id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing authentication with custom param", func(t *testing.T) {
		userID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/profiles/"+userID.String(), nil)
		req = withChiURLParams(req, map[string]string{"user_id": userID.String()})
		// No user in context

		w := httptest.NewRecorder()

		middleware := RequireSameUserWithCustomParam("user_id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestRequireOwnership tests the RequireOwnership middleware
func TestRequireOwnership(t *testing.T) {
	t.Run("Authorized access with custom owner check", func(t *testing.T) {
		userID := uuid.New()
		ownerCheckFunc := func(r *http.Request) (uuid.UUID, error) {
			// Simulating fetching resource owner from database
			resourceID := chi.URLParam(r, "resource_id")
			if resourceID == "123" {
				return userID, nil
			}
			return uuid.Nil, errors.New("resource not found")
		}

		req := httptest.NewRequest(http.MethodGet, "/resources/123", nil)
		req = withChiURLParams(req, map[string]string{"resource_id": "123"})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handlerCalled := false
		middleware := RequireOwnership(ownerCheckFunc)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware(handler).ServeHTTP(w, req)

		assert.True(t, handlerCalled)
	})

	t.Run("Unauthorized access with custom owner check", func(t *testing.T) {
		authenticatedUserID := uuid.New()
		resourceOwnerID := uuid.New()

		ownerCheckFunc := func(r *http.Request) (uuid.UUID, error) {
			return resourceOwnerID, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/resources/123", nil)
		req = withChiURLParams(req, map[string]string{"resource_id": "123"})
		ctx := auth.SetUserIDInContext(req.Context(), authenticatedUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireOwnership(ownerCheckFunc)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "you can only access your own resources")
	})

	t.Run("Owner check function error", func(t *testing.T) {
		userID := uuid.New()
		ownerCheckFunc := func(r *http.Request) (uuid.UUID, error) {
			return uuid.Nil, errors.New("database error")
		}

		req := httptest.NewRequest(http.MethodGet, "/resources/123", nil)
		req = withChiURLParams(req, map[string]string{"resource_id": "123"})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireOwnership(ownerCheckFunc)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "database error")
	})

	t.Run("Missing authentication", func(t *testing.T) {
		ownerCheckFunc := func(r *http.Request) (uuid.UUID, error) {
			return uuid.New(), nil
		}

		req := httptest.NewRequest(http.MethodGet, "/resources/123", nil)
		req = withChiURLParams(req, map[string]string{"resource_id": "123"})
		// No user in context

		w := httptest.NewRecorder()

		middleware := RequireOwnership(ownerCheckFunc)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authentication required")
	})

	t.Run("Owner check succeeds for matching user", func(t *testing.T) {
		userID := uuid.New()
		handlerCalled := false

		ownerCheckFunc := func(r *http.Request) (uuid.UUID, error) {
			return userID, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/resources/123", nil)
		req = withChiURLParams(req, map[string]string{"resource_id": "123"})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		middleware := RequireOwnership(ownerCheckFunc)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware(handler).ServeHTTP(w, req)

		assert.True(t, handlerCalled, "Handler should have been called")
	})
}

// TestCheckUserIDParam tests the CheckUserIDParam middleware
func TestCheckUserIDParam(t *testing.T) {
	t.Run("Valid UUID continues", func(t *testing.T) {
		validUUID := uuid.New()
		handlerCalled := false

		req := httptest.NewRequest(http.MethodGet, "/users/"+validUUID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": validUUID.String()})

		w := httptest.NewRecorder()

		middleware := CheckUserIDParam("id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware(handler).ServeHTTP(w, req)

		assert.True(t, handlerCalled, "Handler should have been called")
	})

	t.Run("Missing parameter returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)

		w := httptest.NewRecorder()

		middleware := CheckUserIDParam("id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "user ID parameter is required")
	})

	t.Run("Invalid UUID format returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid", nil)
		req = withChiURLParams(req, map[string]string{"id": "not-a-uuid"})

		w := httptest.NewRecorder()

		middleware := CheckUserIDParam("id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user ID format")
	})

	t.Run("Custom parameter name", func(t *testing.T) {
		validUUID := uuid.New()
		handlerCalled := false

		req := httptest.NewRequest(http.MethodGet, "/profiles/"+validUUID.String(), nil)
		req = withChiURLParams(req, map[string]string{"user_id": validUUID.String()})

		w := httptest.NewRecorder()

		middleware := CheckUserIDParam("user_id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware(handler).ServeHTTP(w, req)

		assert.True(t, handlerCalled)
	})

	t.Run("Empty UUID string returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/", nil)
		req = withChiURLParams(req, map[string]string{"id": ""})

		w := httptest.NewRecorder()

		middleware := CheckUserIDParam("id")
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		})

		middleware(handler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "user ID parameter is required")
	})
}
