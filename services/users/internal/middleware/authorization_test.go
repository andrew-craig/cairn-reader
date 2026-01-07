package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestRequireSameUser tests the RequireSameUser middleware
func TestRequireSameUser(t *testing.T) {
	t.Run("Authorized access (same user)", func(t *testing.T) {
		userID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)

		// Set user ID in context (from JWTAuth middleware)
		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "id", Value: userID.String()}}

		handlerCalled := false
		middleware := RequireSameUser()
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.False(t, c.IsAborted())
		assert.True(t, handlerCalled)
	})

	t.Run("Unauthorized access (different user)", func(t *testing.T) {
		authenticatedUserID := uuid.New()
		requestedUserID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/"+requestedUserID.String(), nil)

		// Set different user ID in context
		c.Set(string(auth.UserIDContextKey), authenticatedUserID)
		c.Params = gin.Params{{Key: "id", Value: requestedUserID.String()}}

		middleware := RequireSameUser()
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "you can only access your own resources")
	})

	t.Run("Missing authentication (401)", func(t *testing.T) {
		requestedUserID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/"+requestedUserID.String(), nil)

		// Don't set user ID in context
		c.Params = gin.Params{{Key: "id", Value: requestedUserID.String()}}

		middleware := RequireSameUser()
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authentication required")
	})

	t.Run("Missing ID parameter (400)", func(t *testing.T) {
		userID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users", nil)

		c.Set(string(auth.UserIDContextKey), userID)
		// No :id parameter

		middleware := RequireSameUser()
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "user ID parameter is required")
	})

	t.Run("Invalid UUID format (400)", func(t *testing.T) {
		userID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/invalid-uuid", nil)

		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		middleware := RequireSameUser()
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user ID format")
	})

	t.Run("Request continues when authorized", func(t *testing.T) {
		userID := uuid.New()
		handlerCalled := false

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)

		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "id", Value: userID.String()}}

		middleware := RequireSameUser()
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled, "Handler should have been called")
	})
}

// TestRequireSameUserWithCustomParam tests the RequireSameUserWithCustomParam middleware
func TestRequireSameUserWithCustomParam(t *testing.T) {
	t.Run("Custom parameter name handling", func(t *testing.T) {
		userID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/profiles/"+userID.String(), nil)

		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

		handlerCalled := false
		middleware := RequireSameUserWithCustomParam("user_id")
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.False(t, c.IsAborted())
		assert.True(t, handlerCalled)
	})

	t.Run("Unauthorized with custom param", func(t *testing.T) {
		authenticatedUserID := uuid.New()
		requestedUserID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/profiles/"+requestedUserID.String(), nil)

		c.Set(string(auth.UserIDContextKey), authenticatedUserID)
		c.Params = gin.Params{{Key: "user_id", Value: requestedUserID.String()}}

		middleware := RequireSameUserWithCustomParam("user_id")
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Missing custom parameter", func(t *testing.T) {
		userID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/profiles", nil)

		c.Set(string(auth.UserIDContextKey), userID)

		middleware := RequireSameUserWithCustomParam("user_id")
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid UUID in custom parameter", func(t *testing.T) {
		userID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/profiles/not-a-uuid", nil)

		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "user_id", Value: "not-a-uuid"}}

		middleware := RequireSameUserWithCustomParam("user_id")
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing authentication with custom param", func(t *testing.T) {
		userID := uuid.New()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/profiles/"+userID.String(), nil)

		// No user in context
		c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

		middleware := RequireSameUserWithCustomParam("user_id")
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestRequireOwnership tests the RequireOwnership middleware
func TestRequireOwnership(t *testing.T) {
	t.Run("Authorized access with custom owner check", func(t *testing.T) {
		userID := uuid.New()
		ownerCheckFunc := func(c *gin.Context) (uuid.UUID, error) {
			// Simulating fetching resource owner from database
			resourceID := c.Param("resource_id")
			if resourceID == "123" {
				return userID, nil
			}
			return uuid.Nil, errors.New("resource not found")
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/resources/123", nil)

		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "resource_id", Value: "123"}}

		handlerCalled := false
		middleware := RequireOwnership(ownerCheckFunc)
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.False(t, c.IsAborted())
		assert.True(t, handlerCalled)
	})

	t.Run("Unauthorized access with custom owner check", func(t *testing.T) {
		authenticatedUserID := uuid.New()
		resourceOwnerID := uuid.New()

		ownerCheckFunc := func(c *gin.Context) (uuid.UUID, error) {
			return resourceOwnerID, nil
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/resources/123", nil)

		c.Set(string(auth.UserIDContextKey), authenticatedUserID)
		c.Params = gin.Params{{Key: "resource_id", Value: "123"}}

		middleware := RequireOwnership(ownerCheckFunc)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "you can only access your own resources")
	})

	t.Run("Owner check function error", func(t *testing.T) {
		userID := uuid.New()
		ownerCheckFunc := func(c *gin.Context) (uuid.UUID, error) {
			return uuid.Nil, errors.New("database error")
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/resources/123", nil)

		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "resource_id", Value: "123"}}

		middleware := RequireOwnership(ownerCheckFunc)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "database error")
	})

	t.Run("Missing authentication", func(t *testing.T) {
		ownerCheckFunc := func(c *gin.Context) (uuid.UUID, error) {
			return uuid.New(), nil
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/resources/123", nil)

		// No user in context
		c.Params = gin.Params{{Key: "resource_id", Value: "123"}}

		middleware := RequireOwnership(ownerCheckFunc)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authentication required")
	})

	t.Run("Owner check succeeds for matching user", func(t *testing.T) {
		userID := uuid.New()
		handlerCalled := false

		ownerCheckFunc := func(c *gin.Context) (uuid.UUID, error) {
			return userID, nil
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/resources/123", nil)

		c.Set(string(auth.UserIDContextKey), userID)
		c.Params = gin.Params{{Key: "resource_id", Value: "123"}}

		middleware := RequireOwnership(ownerCheckFunc)
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled, "Handler should have been called")
	})
}

// TestCheckUserIDParam tests the CheckUserIDParam middleware
func TestCheckUserIDParam(t *testing.T) {
	t.Run("Valid UUID continues", func(t *testing.T) {
		validUUID := uuid.New()
		handlerCalled := false

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/"+validUUID.String(), nil)

		c.Params = gin.Params{{Key: "id", Value: validUUID.String()}}

		middleware := CheckUserIDParam("id")
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.False(t, c.IsAborted())
		assert.True(t, handlerCalled, "Handler should have been called")
	})

	t.Run("Missing parameter returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users", nil)

		middleware := CheckUserIDParam("id")
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "user ID parameter is required")
	})

	t.Run("Invalid UUID format returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/not-a-uuid", nil)

		c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

		middleware := CheckUserIDParam("id")
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user ID format")
	})

	t.Run("Custom parameter name", func(t *testing.T) {
		validUUID := uuid.New()
		handlerCalled := false

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/profiles/"+validUUID.String(), nil)

		c.Params = gin.Params{{Key: "user_id", Value: validUUID.String()}}

		middleware := CheckUserIDParam("user_id")
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.False(t, c.IsAborted())
		assert.True(t, handlerCalled)
	})

	t.Run("Empty UUID string returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/users/", nil)

		c.Params = gin.Params{{Key: "id", Value: ""}}

		middleware := CheckUserIDParam("id")
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "user ID parameter is required")
	})
}
