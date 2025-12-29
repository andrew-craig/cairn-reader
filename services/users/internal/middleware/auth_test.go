package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn/services/users/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Helper function to create a test JWT manager
func createTestJWTManager(t *testing.T) *auth.JWTManager {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwtManager := auth.NewJWTManager(privateKey, &privateKey.PublicKey, 60*time.Minute)
	return jwtManager
}

// TestJWTAuth tests the JWTAuth middleware
func TestJWTAuth(t *testing.T) {
	jwtManager := createTestJWTManager(t)

	t.Run("Valid token allows request", func(t *testing.T) {
		// Create a valid token
		userID := uuid.New()
		token, err := jwtManager.GenerateToken(userID)
		require.NoError(t, err)

		// Create test request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		// Create handler that checks if user ID is in context
		handlerCalled := false
		middleware := JWTAuth(jwtManager)
		handler := func(c *gin.Context) {
			handlerCalled = true
			// Verify user ID is in context
			storedUserID, err := GetUserIDFromContext(c)
			assert.NoError(t, err)
			assert.Equal(t, userID, storedUserID)
		}

		// Execute middleware
		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled, "Handler should have been called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Missing Authorization header returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)

		middleware := JWTAuth(jwtManager)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing authorization header")
	})

	t.Run("Invalid header format returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "InvalidFormat")

		middleware := JWTAuth(jwtManager)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid authorization header")
	})

	t.Run("Empty token returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer ")

		middleware := JWTAuth(jwtManager)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing token")
	})

	t.Run("Expired token returns 401", func(t *testing.T) {
		// Create a JWT manager with negative expiry using the SAME keys as jwtManager
		// First, extract the keys from jwtManager by creating a token with it
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Create both managers with same keys
		expiredJWT := auth.NewJWTManager(privateKey, &privateKey.PublicKey, -1*time.Hour)
		validatingJWT := auth.NewJWTManager(privateKey, &privateKey.PublicKey, 60*time.Minute)

		userID := uuid.New()
		token, err := expiredJWT.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		middleware := JWTAuth(validatingJWT)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "token has expired")
	})

	t.Run("Invalid signature returns 401", func(t *testing.T) {
		// Create token with different JWT manager (different keys)
		differentJWTManager := createTestJWTManager(t)
		userID := uuid.New()
		token, err := differentJWTManager.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		middleware := JWTAuth(jwtManager)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid token signature")
	})

	t.Run("Invalid issuer returns 401", func(t *testing.T) {
		// Create JWT manager with custom issuer using SAME keys
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Token generator with different issuer
		tokenJWT := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
			Expiry:     60 * time.Minute,
			Issuer:     "different-issuer",
			Audience:   "cairn-api",
		})

		// Validator with expected issuer (but same keys)
		validatorJWT := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
			Expiry:     60 * time.Minute,
			Issuer:     "cairn-user-service",
			Audience:   "cairn-api",
		})

		userID := uuid.New()
		token, err := tokenJWT.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		// Use the validator with expected issuer
		middleware := JWTAuth(validatorJWT)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid token issuer")
	})

	t.Run("Invalid audience returns 401", func(t *testing.T) {
		// Create JWT manager with custom audience using SAME keys
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Token generator with different audience
		tokenJWT := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
			Expiry:     60 * time.Minute,
			Issuer:     "cairn-user-service",
			Audience:   "different-audience",
		})

		// Validator with expected audience (but same keys)
		validatorJWT := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
			Expiry:     60 * time.Minute,
			Issuer:     "cairn-user-service",
			Audience:   "cairn-api",
		})

		userID := uuid.New()
		token, err := tokenJWT.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		// Use the validator with expected audience
		middleware := JWTAuth(validatorJWT)
		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid token audience")
	})

	t.Run("User ID stored in context", func(t *testing.T) {
		userID := uuid.New()
		token, err := jwtManager.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		middleware := JWTAuth(jwtManager)
		middleware(c)

		// Check user ID in context
		storedUserID, err := GetUserIDFromContext(c)
		require.NoError(t, err)
		assert.Equal(t, userID, storedUserID)
	})

	t.Run("Case insensitive Bearer prefix", func(t *testing.T) {
		userID := uuid.New()
		token, err := jwtManager.GenerateToken(userID)
		require.NoError(t, err)

		testCases := []string{
			"Bearer " + token,
			"bearer " + token,
			"BEARER " + token,
			"BeArEr " + token,
		}

		for _, authHeader := range testCases {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
			c.Request.Header.Set("Authorization", authHeader)

			handlerCalled := false
			middleware := JWTAuth(jwtManager)
			handler := func(c *gin.Context) {
				handlerCalled = true
			}

			middleware(c)
			if !c.IsAborted() {
				handler(c)
			}

			assert.True(t, handlerCalled, "Handler should have been called for: %s", authHeader)
		}
	})
}

// TestGetUserIDFromContext tests the GetUserIDFromContext function
func TestGetUserIDFromContext(t *testing.T) {
	t.Run("Valid user ID retrieval", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID := uuid.New()
		c.Set(UserIDKey, userID)

		retrievedID, err := GetUserIDFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, userID, retrievedID)
	})

	t.Run("Missing user ID in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		_, err := GetUserIDFromContext(c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID not found in context")
	})

	t.Run("Invalid type in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(UserIDKey, "not-a-uuid")

		_, err := GetUserIDFromContext(c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID type")
	})
}

// TestRequireAuth tests the RequireAuth alias
func TestRequireAuth(t *testing.T) {
	jwtManager := createTestJWTManager(t)

	t.Run("RequireAuth is alias for JWTAuth", func(t *testing.T) {
		userID := uuid.New()
		token, err := jwtManager.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		handlerCalled := false
		middleware := RequireAuth(jwtManager)
		handler := func(c *gin.Context) {
			handlerCalled = true
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestOptionalAuth tests the OptionalAuth middleware
func TestOptionalAuth(t *testing.T) {
	jwtManager := createTestJWTManager(t)

	t.Run("Valid token sets context", func(t *testing.T) {
		userID := uuid.New()
		token, err := jwtManager.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		handlerCalled := false
		middleware := OptionalAuth(jwtManager)
		handler := func(c *gin.Context) {
			handlerCalled = true
			// User should be in context
			storedUserID, err := GetUserIDFromContext(c)
			assert.NoError(t, err)
			assert.Equal(t, userID, storedUserID)
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled)
		assert.False(t, c.IsAborted())
	})

	t.Run("Missing token continues without authentication", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)

		handlerCalled := false
		middleware := OptionalAuth(jwtManager)
		handler := func(c *gin.Context) {
			handlerCalled = true
			// User should not be in context
			_, err := GetUserIDFromContext(c)
			assert.Error(t, err)
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled)
		assert.False(t, c.IsAborted())
	})

	t.Run("Invalid token continues without authentication", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer invalid-token")

		handlerCalled := false
		middleware := OptionalAuth(jwtManager)
		handler := func(c *gin.Context) {
			handlerCalled = true
			// User should not be in context
			_, err := GetUserIDFromContext(c)
			assert.Error(t, err)
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled)
		assert.False(t, c.IsAborted())
	})

	t.Run("User ID in context when valid", func(t *testing.T) {
		userID := uuid.New()
		token, err := jwtManager.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		middleware := OptionalAuth(jwtManager)
		middleware(c)

		storedUserID, err := GetUserIDFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, userID, storedUserID)
	})

	t.Run("Expired token continues without authentication", func(t *testing.T) {
		// Create an expired token
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		expiredJWT := auth.NewJWTManager(privateKey, &privateKey.PublicKey, -1*time.Hour)

		userID := uuid.New()
		token, err := expiredJWT.GenerateToken(userID)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		handlerCalled := false
		middleware := OptionalAuth(jwtManager)
		handler := func(c *gin.Context) {
			handlerCalled = true
			_, err := GetUserIDFromContext(c)
			assert.Error(t, err)
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled)
		assert.False(t, c.IsAborted())
	})
}

// TestIsAuthenticated tests the IsAuthenticated function
func TestIsAuthenticated(t *testing.T) {
	t.Run("Returns true when authenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID := uuid.New()
		c.Set(UserIDKey, userID)

		assert.True(t, IsAuthenticated(c))
	})

	t.Run("Returns false when not authenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		assert.False(t, IsAuthenticated(c))
	})
}

// TestMustGetUserID tests the MustGetUserID function
func TestMustGetUserID(t *testing.T) {
	t.Run("Returns user ID when present", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID := uuid.New()
		c.Set(UserIDKey, userID)

		retrievedID := MustGetUserID(c)
		assert.Equal(t, userID, retrievedID)
	})

	t.Run("Panics when missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		assert.Panics(t, func() {
			MustGetUserID(c)
		})
	})

	t.Run("Panics with correct message", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		defer func() {
			r := recover()
			assert.NotNil(t, r)
			assert.Contains(t, r, "user ID not found in context")
		}()

		MustGetUserID(c)
	})
}
