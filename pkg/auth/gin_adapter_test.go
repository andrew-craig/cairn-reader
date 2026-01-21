package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestGetUserIDFromGinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	t.Run("with user ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(string(UserIDContextKey), userID)

		retrievedID, err := GetUserIDFromGinContext(c)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retrievedID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, retrievedID)
		}
	})

	t.Run("without user ID returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		retrievedID, err := GetUserIDFromGinContext(c)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err != ErrUserIDNotFound {
			t.Errorf("Expected ErrUserIDNotFound, got %v", err)
		}
		if retrievedID != uuid.Nil {
			t.Errorf("Expected nil UUID, got %s", retrievedID)
		}
	})

	t.Run("with invalid type returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(string(UserIDContextKey), "invalid-type")

		retrievedID, err := GetUserIDFromGinContext(c)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if retrievedID != uuid.Nil {
			t.Errorf("Expected nil UUID, got %s", retrievedID)
		}
	})
}

func TestMustGetUserIDFromGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	t.Run("with user ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(string(UserIDContextKey), userID)

		retrievedID := MustGetUserIDFromGin(c)
		if retrievedID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, retrievedID)
		}
	})

	t.Run("without user ID panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected MustGetUserIDFromGin to panic")
			}
		}()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		MustGetUserIDFromGin(c)
	})
}

func TestIsAuthenticatedInGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	t.Run("authenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(string(UserIDContextKey), userID)

		if !IsAuthenticatedInGin(c) {
			t.Error("Expected IsAuthenticatedInGin to return true")
		}
	})

	t.Run("not authenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		if IsAuthenticatedInGin(c) {
			t.Error("Expected IsAuthenticatedInGin to return false")
		}
	})
}

func TestGinMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a test handler that uses GetUserIDFromGinContext
	testHandler := func(c *gin.Context) {
		userID, err := GetUserIDFromGinContext(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "auth context error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID.String()})
	}

	t.Run("handler with GetUserIDFromGinContext returns error when auth missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		testHandler(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("handler with GetUserIDFromGinContext succeeds when auth present", func(t *testing.T) {
		userID := uuid.New()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Set(string(UserIDContextKey), userID)

		testHandler(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}
