package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn/services/users/internal/auth"
	"github.com/andrew-craig/cairn/services/users/internal/database"
	"github.com/andrew-craig/cairn/services/users/internal/middleware"
	"github.com/andrew-craig/cairn/services/users/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestAuthHandler creates a test auth handler with all dependencies
func setupTestAuthHandler(t *testing.T) (*AuthHandler, *database.DB, *auth.JWTManager, func()) {
	db := setupTestDB(t)

	// Create password hasher (bcrypt cost 10 for faster tests)
	passwordHasher := auth.NewPasswordHasher(10)

	// Create JWT manager with test keys
	privateKey, publicKey := generateTestRSAKeys(t)

	jwtManager := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Expiry:     60 * time.Minute,
		Issuer:     "test-issuer",
		Audience:   "test-audience",
	})

	// Create refresh token service
	refreshTokenRepo := database.NewRefreshTokenRepository(db)
	refreshTokenService := auth.NewRefreshTokenService(refreshTokenRepo, 30*24*time.Hour)

	// Create user repository
	userRepo := database.NewUserRepository(db)

	// Create auth service
	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            userRepo,
		RefreshTokenService: refreshTokenService,
		JWTManager:          jwtManager,
		PasswordHasher:      passwordHasher,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	handler := NewAuthHandler(authService)

	cleanup := func() {
		db.Close()
	}

	return handler, db, jwtManager, cleanup
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *database.DB {
	cfg := &database.Config{
		Host:            getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            getEnvOrDefault("TEST_DB_PORT", "5432"),
		User:            getEnvOrDefault("TEST_DB_USER", "postgres"),
		Password:        getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		Database:        getEnvOrDefault("TEST_DB_NAME", "cairn_test"),
		SSLMode:         getEnvOrDefault("TEST_DB_SSLMODE", "disable"),
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db, err := database.New(cfg)
	if err != nil {
		t.Skip("Database not available for testing:", err)
	}

	return db
}

func getEnvOrDefault(key, defaultValue string) string {
	// In real implementation, use os.Getenv
	return defaultValue
}

// generateTestRSAKeys generates test RSA key pairs for JWT
func generateTestRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return privateKey, &privateKey.PublicKey
}

// cleanupTestUser removes a test user from the database
func cleanupTestUser(t *testing.T, db *database.DB, userID uuid.UUID) {
	_, err := db.Pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test refresh tokens: %v", err)
	}

	_, err = db.Pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test user: %v", err)
	}
}

// TestRegister tests the POST /auth/register endpoint
func TestRegister(t *testing.T) {
	handler, db, _, cleanup := setupTestAuthHandler(t)
	defer cleanup()

	t.Run("Valid registration", func(t *testing.T) {
		reqBody := RegisterRequest{
			Email:    "test@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp services.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.NotEmpty(t, resp.User.ID)
		assert.Equal(t, reqBody.Email, resp.User.Email)

		// Cleanup
		cleanupTestUser(t, db, resp.User.ID)
	})

	t.Run("Invalid JSON body returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("invalid json"))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request")
	})

	t.Run("Missing email returns 400", func(t *testing.T) {
		reqBody := map[string]string{
			"password": "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request")
	})

	t.Run("Missing password returns 400", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "test@example.com",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request")
	})

	t.Run("Weak password returns 400", func(t *testing.T) {
		reqBody := RegisterRequest{
			Email:    "test@example.com",
			Password: "weak",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "password")
	})

	t.Run("Duplicate email returns 409", func(t *testing.T) {
		// Create initial user
		reqBody := RegisterRequest{
			Email:    "duplicate@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp services.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		defer cleanupTestUser(t, db, resp.User.ID)

		// Try to create duplicate
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "already exists")
	})

	t.Run("Response structure validation", func(t *testing.T) {
		reqBody := RegisterRequest{
			Email:    "response@example.com",
			Password: "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Register(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp services.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Greater(t, resp.ExpiresIn, int64(0))
		assert.NotEmpty(t, resp.User.ID)
		assert.Equal(t, reqBody.Email, resp.User.Email)
		assert.NotZero(t, resp.User.CreatedAt)

		// Cleanup
		cleanupTestUser(t, db, resp.User.ID)
	})
}

// TestRegisterMobile tests the POST /auth/register/mobile endpoint
func TestRegisterMobile(t *testing.T) {
	handler, db, _, cleanup := setupTestAuthHandler(t)
	defer cleanup()

	t.Run("Valid mobile registration", func(t *testing.T) {
		reqBody := RegisterMobileRequest{
			ExpoDeviceID: "test-device-123",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Header.Set("User-Agent", "Test Device")
		c.Request.RemoteAddr = "127.0.0.1:12345"

		handler.RegisterMobile(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp services.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.NotEmpty(t, resp.User.ID)

		// Cleanup
		cleanupTestUser(t, db, resp.User.ID)
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register/mobile", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.RegisterMobile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request")
	})

	t.Run("Missing device ID", func(t *testing.T) {
		reqBody := map[string]string{}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.RegisterMobile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request")
	})

	t.Run("Duplicate device ID returns 409", func(t *testing.T) {
		reqBody := RegisterMobileRequest{
			ExpoDeviceID: "duplicate-device",
		}
		body, _ := json.Marshal(reqBody)

		// Create initial user
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.RegisterMobile(c)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp services.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		defer cleanupTestUser(t, db, resp.User.ID)

		// Try to create duplicate
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.RegisterMobile(c)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "already exists")
	})

	t.Run("Device info and IP capture", func(t *testing.T) {
		reqBody := RegisterMobileRequest{
			ExpoDeviceID: "device-with-info",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Header.Set("User-Agent", "Test Device Agent")
		c.Request.RemoteAddr = "192.168.1.1:12345"

		handler.RegisterMobile(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp services.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		defer cleanupTestUser(t, db, resp.User.ID)
	})
}

// TestLogin tests the POST /auth/login endpoint
func TestLogin(t *testing.T) {
	handler, db, _, cleanup := setupTestAuthHandler(t)
	defer cleanup()

	// Create a test user
	email := "login@example.com"
	password := "SecurePass123!"

	reqBody := RegisterRequest{Email: email, Password: password}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)
	require.Equal(t, http.StatusCreated, w.Code)

	var registerResp services.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &registerResp)
	defer cleanupTestUser(t, db, registerResp.User.ID)

	t.Run("Valid email/password login", func(t *testing.T) {
		loginBody := LoginRequest{
			Email:    email,
			Password: password,
		}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp services.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, email, resp.User.Email)
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid request")
	})

	t.Run("Missing credentials returns 400", func(t *testing.T) {
		loginBody := map[string]string{
			"email": email,
		}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid credentials returns 401", func(t *testing.T) {
		loginBody := LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "WrongPassword123!",
		}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email or password")
	})

	t.Run("Incorrect password returns 401", func(t *testing.T) {
		loginBody := LoginRequest{
			Email:    email,
			Password: "WrongPassword123!",
		}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid email or password")
	})

	t.Run("Response structure validation", func(t *testing.T) {
		loginBody := LoginRequest{
			Email:    email,
			Password: password,
		}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp services.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Greater(t, resp.ExpiresIn, int64(0))
		assert.Equal(t, email, resp.User.Email)
	})
}

// TestLoginMobile tests the POST /auth/login/mobile endpoint
func TestLoginMobile(t *testing.T) {
	handler, db, _, cleanup := setupTestAuthHandler(t)
	defer cleanup()

	// Create a mobile-only user
	deviceID := "mobile-login-device"
	reqBody := RegisterMobileRequest{ExpoDeviceID: deviceID}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register/mobile", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RegisterMobile(c)
	require.Equal(t, http.StatusCreated, w.Code)

	var registerResp services.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &registerResp)
	defer cleanupTestUser(t, db, registerResp.User.ID)

	t.Run("Valid device ID login", func(t *testing.T) {
		loginBody := LoginMobileRequest{
			ExpoDeviceID: deviceID,
		}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.LoginMobile(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp services.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login/mobile", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.LoginMobile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing device ID", func(t *testing.T) {
		loginBody := map[string]string{}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.LoginMobile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid device ID returns 401", func(t *testing.T) {
		loginBody := LoginMobileRequest{
			ExpoDeviceID: "nonexistent-device",
		}
		body, _ := json.Marshal(loginBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/login/mobile", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.LoginMobile(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid device ID")
	})
}

// TestRefresh tests the POST /auth/refresh endpoint
func TestRefresh(t *testing.T) {
	handler, db, _, cleanup := setupTestAuthHandler(t)
	defer cleanup()

	// Create a test user and get tokens
	email := "refresh@example.com"
	password := "SecurePass123!"

	reqBody := RegisterRequest{Email: email, Password: password}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)
	require.Equal(t, http.StatusCreated, w.Code)

	var registerResp services.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &registerResp)
	defer cleanupTestUser(t, db, registerResp.User.ID)

	t.Run("Valid token refresh", func(t *testing.T) {
		refreshBody := RefreshRequest{
			RefreshToken: registerResp.RefreshToken,
		}
		body, _ := json.Marshal(refreshBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Refresh(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp services.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		// New refresh token should be different from old one
		assert.NotEqual(t, registerResp.RefreshToken, resp.RefreshToken)
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Refresh(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing refresh token returns 400", func(t *testing.T) {
		refreshBody := map[string]string{}
		body, _ := json.Marshal(refreshBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Refresh(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid refresh token returns 401", func(t *testing.T) {
		refreshBody := RefreshRequest{
			RefreshToken: "invalid-token",
		}
		body, _ := json.Marshal(refreshBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Refresh(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired")
	})
}

// TestLogout tests the POST /auth/logout endpoint
func TestLogout(t *testing.T) {
	handler, db, _, cleanup := setupTestAuthHandler(t)
	defer cleanup()

	// Create a test user and get tokens
	email := "logout@example.com"
	password := "SecurePass123!"

	reqBody := RegisterRequest{Email: email, Password: password}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)
	require.Equal(t, http.StatusCreated, w.Code)

	var registerResp services.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &registerResp)
	defer cleanupTestUser(t, db, registerResp.User.ID)

	t.Run("Valid logout", func(t *testing.T) {
		logoutBody := LogoutRequest{
			RefreshToken: registerResp.RefreshToken,
		}
		body, _ := json.Marshal(logoutBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/logout", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Logout(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "successfully logged out")
	})

	t.Run("Missing refresh token", func(t *testing.T) {
		logoutBody := map[string]string{}
		body, _ := json.Marshal(logoutBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/logout", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Logout(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Already revoked token returns success", func(t *testing.T) {
		// Use an invalid token - logout should not error
		logoutBody := LogoutRequest{
			RefreshToken: "already-revoked-token",
		}
		body, _ := json.Marshal(logoutBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/logout", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Logout(c)

		// Should succeed since logout is idempotent
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestLogoutAll tests the POST /auth/logout-all endpoint
func TestLogoutAll(t *testing.T) {
	handler, db, jwtManager, cleanup := setupTestAuthHandler(t)
	defer cleanup()

	// Create a test user and get tokens
	email := "logoutall@example.com"
	password := "SecurePass123!"

	reqBody := RegisterRequest{Email: email, Password: password}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)
	require.Equal(t, http.StatusCreated, w.Code)

	var registerResp services.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &registerResp)
	defer cleanupTestUser(t, db, registerResp.User.ID)

	t.Run("Valid logout all", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/logout-all", nil)
		c.Request.Header.Set("Authorization", "Bearer "+registerResp.AccessToken)

		// Apply JWT middleware to set user ID in context
		middleware.JWTAuth(jwtManager)(c)
		if !c.IsAborted() {
			handler.LogoutAll(c)
		}

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "all devices")
	})

	t.Run("Authentication required", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/logout-all", nil)
		// No Authorization header

		handler.LogoutAll(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "authentication required")
	})

	t.Run("User ID from JWT token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/logout-all", nil)
		c.Request.Header.Set("Authorization", "Bearer "+registerResp.AccessToken)

		// Apply JWT middleware
		middleware.JWTAuth(jwtManager)(c)
		if !c.IsAborted() {
			handler.LogoutAll(c)
		}

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
