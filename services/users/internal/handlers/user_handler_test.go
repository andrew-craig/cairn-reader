package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalAuth "github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
)

// setupTestUserHandler creates a test user handler with all dependencies
func setupTestUserHandler(t *testing.T) (*UserHandler, *database.DB, *internalAuth.JWTManager, func()) {
	db := setupTestDB(t)

	// Create password hasher (bcrypt cost 10 for faster tests)
	passwordHasher := internalAuth.NewPasswordHasher(10)

	// Create JWT manager with test keys
	privateKey, publicKey := generateTestRSAKeys(t)

	jwtManager := internalAuth.NewJWTManagerWithConfig(internalAuth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Expiry:     60 * time.Minute,
		Issuer:     "test-issuer",
		Audience:   "test-audience",
	})

	// Create refresh token repository
	refreshTokenRepo := database.NewRefreshTokenRepository(db)

	// Create user repository
	userRepo := database.NewUserRepository(db)

	// Create user service
	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          userRepo,
		RefreshTokenRepo:  refreshTokenRepo,
		PasswordHasher:    passwordHasher,
		PasswordMinLength: 8,
		RequireComplexity: true,
	})

	handler := NewUserHandler(userService)

	cleanup := func() {
		db.Close()
	}

	return handler, db, jwtManager, cleanup
}

// createTestUser creates a user for testing
func createTestUser(t *testing.T, db *database.DB, email string, password string) (uuid.UUID, string) {
	userRepo := database.NewUserRepository(db)
	passwordHasher := internalAuth.NewPasswordHasher(10)

	hashedPassword, err := passwordHasher.HashPassword(password)
	require.NoError(t, err)

	user, err := userRepo.CreateUser(context.Background(), email, hashedPassword)
	require.NoError(t, err)

	return user.ID, hashedPassword
}

// createTestMobileUser creates a mobile-only user for testing
func createTestMobileUser(t *testing.T, db *database.DB, deviceID string) uuid.UUID {
	userRepo := database.NewUserRepository(db)

	user, err := userRepo.CreateMobileUser(context.Background(), deviceID)
	require.NoError(t, err)

	return user.ID
}

// withChiURLParams adds Chi URL parameters to a request context
func withChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// TestGetUser tests the GET /users/:id endpoint
func TestGetUser(t *testing.T) {
	handler, db, jwtManager, cleanup := setupTestUserHandler(t)
	defer cleanup()

	// Create a test user
	userID, _ := createTestUser(t, db, "getuser@example.com", "SecurePass123!")
	defer cleanupTestUser(t, db, userID)

	// Generate access token for the user
	accessToken, err := jwtManager.GenerateToken(userID)
	require.NoError(t, err)

	t.Run("Valid user retrieval", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Add Chi URL params and user ID to context
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var user models.User
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)

		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "getuser@example.com", *user.Email)
	})

	t.Run("Authentication required returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		// No Authorization header, no user ID in context

		w := httptest.NewRecorder()
		handler.GetUser(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authentication required")
	})

	t.Run("Authorization required returns 403", func(t *testing.T) {
		// Create another user
		otherUserID, _ := createTestUser(t, db, "otheruser@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, otherUserID)

		req := httptest.NewRequest(http.MethodGet, "/users/"+otherUserID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Add Chi URL params with otherUserID but authenticate as userID
		req = withChiURLParams(req, map[string]string{"id": otherUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetUser(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "only access your own")
	})

	t.Run("Non-existent user returns 404", func(t *testing.T) {
		nonExistentID := uuid.New()
		nonExistentToken, _ := jwtManager.GenerateToken(nonExistentID)

		req := httptest.NewRequest(http.MethodGet, "/users/"+nonExistentID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+nonExistentToken)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), nonExistentID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetUser(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "not found")
	})

	t.Run("Invalid user ID format returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/invalid-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": "invalid-uuid"})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user ID format")
	})

	t.Run("Response structure validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var user models.User
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)

		assert.NotEmpty(t, user.ID)
		assert.NotNil(t, user.Email)
		assert.NotZero(t, user.CreatedAt)
	})
}

// TestUpdateUser tests the PATCH /users/:id endpoint
func TestUpdateUser(t *testing.T) {
	handler, db, jwtManager, cleanup := setupTestUserHandler(t)
	defer cleanup()

	// Create a test user
	userID, _ := createTestUser(t, db, "updateuser@example.com", "SecurePass123!")
	defer cleanupTestUser(t, db, userID)

	// Generate access token
	accessToken, err := jwtManager.GenerateToken(userID)
	require.NoError(t, err)

	t.Run("Valid user update", func(t *testing.T) {
		newEmail := "updated@example.com"
		updateReq := UpdateUserRequest{
			Email: &newEmail,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPatch, "/users/"+userID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpdateUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var user models.User
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)

		assert.Equal(t, newEmail, *user.Email)
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/users/"+userID.String(), bytes.NewBufferString("invalid"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpdateUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request")
	})

	t.Run("No fields to update returns 400", func(t *testing.T) {
		updateReq := UpdateUserRequest{
			Email: nil,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPatch, "/users/"+userID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpdateUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "no fields to update")
	})

	t.Run("Authentication required", func(t *testing.T) {
		newEmail := "updated@example.com"
		updateReq := UpdateUserRequest{
			Email: &newEmail,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPatch, "/users/"+userID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		// No Authorization header, no user ID in context

		w := httptest.NewRecorder()
		handler.UpdateUser(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Authorization required returns 403", func(t *testing.T) {
		// Create another user
		otherUserID, _ := createTestUser(t, db, "otheruser2@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, otherUserID)

		newEmail := "hacked@example.com"
		updateReq := UpdateUserRequest{
			Email: &newEmail,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPatch, "/users/"+otherUserID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": otherUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpdateUser(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Duplicate email returns 409", func(t *testing.T) {
		// Create another user with a different email
		otherUserID, _ := createTestUser(t, db, "existing@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, otherUserID)

		// Try to update first user's email to the second user's email
		duplicateEmail := "existing@example.com"
		updateReq := UpdateUserRequest{
			Email: &duplicateEmail,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPatch, "/users/"+userID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpdateUser(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "already exists")
	})

	t.Run("Non-existent user returns 404", func(t *testing.T) {
		nonExistentID := uuid.New()
		nonExistentToken, _ := jwtManager.GenerateToken(nonExistentID)

		newEmail := "new@example.com"
		updateReq := UpdateUserRequest{
			Email: &newEmail,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPatch, "/users/"+nonExistentID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+nonExistentToken)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), nonExistentID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpdateUser(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestUpgradeAccount tests the POST /users/:id/upgrade endpoint
func TestUpgradeAccount(t *testing.T) {
	handler, db, jwtManager, cleanup := setupTestUserHandler(t)
	defer cleanup()

	// Create a mobile-only user
	deviceID := "upgrade-device-123"
	userID := createTestMobileUser(t, db, deviceID)
	defer cleanupTestUser(t, db, userID)

	// Generate access token
	accessToken, err := jwtManager.GenerateToken(userID)
	require.NoError(t, err)

	t.Run("Valid account upgrade", func(t *testing.T) {
		upgradeReq := UpgradeAccountRequest{
			Email:    "upgraded@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(upgradeReq)

		req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/upgrade", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var user models.User
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)

		assert.Equal(t, "upgraded@example.com", *user.Email)
		assert.NotNil(t, user.PasswordHash)
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/upgrade", bytes.NewBufferString("invalid"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Authentication required", func(t *testing.T) {
		upgradeReq := UpgradeAccountRequest{
			Email:    "upgraded2@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(upgradeReq)

		req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/upgrade", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		// No Authorization header, no user ID in context

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Authorization required", func(t *testing.T) {
		// Create another mobile user
		otherDeviceID := "other-device"
		otherUserID := createTestMobileUser(t, db, otherDeviceID)
		defer cleanupTestUser(t, db, otherUserID)

		upgradeReq := UpgradeAccountRequest{
			Email:    "hacked2@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(upgradeReq)

		req := httptest.NewRequest(http.MethodPost, "/users/"+otherUserID.String()+"/upgrade", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": otherUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Weak password returns 400", func(t *testing.T) {
		// Create a new mobile user for this test
		newDeviceID := "weak-pwd-device"
		newUserID := createTestMobileUser(t, db, newDeviceID)
		defer cleanupTestUser(t, db, newUserID)

		newAccessToken, _ := jwtManager.GenerateToken(newUserID)

		upgradeReq := UpgradeAccountRequest{
			Email:    "weakpwd@example.com",
			Password: "weak",
		}
		body, _ := json.Marshal(upgradeReq)

		req := httptest.NewRequest(http.MethodPost, "/users/"+newUserID.String()+"/upgrade", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+newAccessToken)
		req = withChiURLParams(req, map[string]string{"id": newUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), newUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "password")
	})

	t.Run("Non-mobile account returns 400", func(t *testing.T) {
		// Create an email user
		emailUserID, _ := createTestUser(t, db, "emailuser@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, emailUserID)

		emailAccessToken, _ := jwtManager.GenerateToken(emailUserID)

		upgradeReq := UpgradeAccountRequest{
			Email:    "upgrade-email@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(upgradeReq)

		req := httptest.NewRequest(http.MethodPost, "/users/"+emailUserID.String()+"/upgrade", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+emailAccessToken)
		req = withChiURLParams(req, map[string]string{"id": emailUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), emailUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "not a mobile-only account")
	})

	t.Run("Duplicate email returns 409", func(t *testing.T) {
		// Create an existing user with email
		existingUserID, _ := createTestUser(t, db, "existing2@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, existingUserID)

		// Create a new mobile user
		newDeviceID := "dup-email-device"
		newUserID := createTestMobileUser(t, db, newDeviceID)
		defer cleanupTestUser(t, db, newUserID)

		newAccessToken, _ := jwtManager.GenerateToken(newUserID)

		upgradeReq := UpgradeAccountRequest{
			Email:    "existing2@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(upgradeReq)

		req := httptest.NewRequest(http.MethodPost, "/users/"+newUserID.String()+"/upgrade", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+newAccessToken)
		req = withChiURLParams(req, map[string]string{"id": newUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), newUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Account type verification", func(t *testing.T) {
		// Create a new mobile user
		verifyDeviceID := "verify-device"
		verifyUserID := createTestMobileUser(t, db, verifyDeviceID)
		defer cleanupTestUser(t, db, verifyUserID)

		verifyAccessToken, _ := jwtManager.GenerateToken(verifyUserID)

		upgradeReq := UpgradeAccountRequest{
			Email:    "verify@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(upgradeReq)

		req := httptest.NewRequest(http.MethodPost, "/users/"+verifyUserID.String()+"/upgrade", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+verifyAccessToken)
		req = withChiURLParams(req, map[string]string{"id": verifyUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), verifyUserID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpgradeAccount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var user models.User
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)

		// Should be hybrid account now
		assert.True(t, user.IsHybrid())
		assert.False(t, user.IsMobileOnly())
	})
}

// TestDeleteUser tests the DELETE /users/:id endpoint
func TestDeleteUser(t *testing.T) {
	handler, db, jwtManager, cleanup := setupTestUserHandler(t)
	defer cleanup()

	t.Run("Valid user deletion", func(t *testing.T) {
		// Create a test user
		userID, _ := createTestUser(t, db, "deleteuser@example.com", "SecurePass123!")
		// Note: we don't defer cleanup since we're testing deletion

		accessToken, err := jwtManager.GenerateToken(userID)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.DeleteUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "successfully deleted")
	})

	t.Run("Authentication required", func(t *testing.T) {
		userID, _ := createTestUser(t, db, "deleteuser2@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, userID)

		req := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String(), nil)
		req = withChiURLParams(req, map[string]string{"id": userID.String()})
		// No Authorization header, no user ID in context

		w := httptest.NewRecorder()
		handler.DeleteUser(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Authorization required", func(t *testing.T) {
		userID, _ := createTestUser(t, db, "deleteuser3@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, userID)

		otherUserID, _ := createTestUser(t, db, "deleteuser4@example.com", "SecurePass123!")
		defer cleanupTestUser(t, db, otherUserID)

		accessToken, _ := jwtManager.GenerateToken(userID)

		req := httptest.NewRequest(http.MethodDelete, "/users/"+otherUserID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req = withChiURLParams(req, map[string]string{"id": otherUserID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.DeleteUser(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Non-existent user returns 404", func(t *testing.T) {
		nonExistentID := uuid.New()
		nonExistentToken, _ := jwtManager.GenerateToken(nonExistentID)

		req := httptest.NewRequest(http.MethodDelete, "/users/"+nonExistentID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+nonExistentToken)
		req = withChiURLParams(req, map[string]string{"id": nonExistentID.String()})
		ctx := auth.SetUserIDInContext(req.Context(), nonExistentID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.DeleteUser(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
