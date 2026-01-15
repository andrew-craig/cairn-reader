package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestKeyPair generates a test RSA key pair
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return privateKey, &privateKey.PublicKey
}

func TestNewJWTManager(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	expiry := 60 * time.Minute

	manager := NewJWTManager(privateKey, publicKey, expiry)

	assert.NotNil(t, manager)
	assert.Equal(t, privateKey, manager.privateKey)
	assert.Equal(t, publicKey, manager.publicKey)
	assert.Equal(t, expiry, manager.expiry)
	assert.Equal(t, "cairn-user-service", manager.issuer)
	assert.Equal(t, "cairn-api", manager.audience)
}

func TestNewJWTManagerWithConfig(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)

	tests := []struct {
		name           string
		config         JWTManagerConfig
		expectedIssuer string
		expectedAud    string
	}{
		{
			name: "with custom issuer and audience",
			config: JWTManagerConfig{
				PrivateKey: privateKey,
				PublicKey:  publicKey,
				Expiry:     30 * time.Minute,
				Issuer:     "custom-issuer",
				Audience:   "custom-audience",
			},
			expectedIssuer: "custom-issuer",
			expectedAud:    "custom-audience",
		},
		{
			name: "with default issuer and audience",
			config: JWTManagerConfig{
				PrivateKey: privateKey,
				PublicKey:  publicKey,
				Expiry:     30 * time.Minute,
			},
			expectedIssuer: "cairn-user-service",
			expectedAud:    "cairn-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewJWTManagerWithConfig(tt.config)
			assert.NotNil(t, manager)
			assert.Equal(t, tt.expectedIssuer, manager.issuer)
			assert.Equal(t, tt.expectedAud, manager.audience)
		})
	}
}

func TestJWTGenerateToken(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token can be parsed
	claims, err := manager.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "cairn-user-service", claims.Issuer)
	assert.Contains(t, claims.Audience, "cairn-api")
	assert.Equal(t, userID.String(), claims.Subject)
}

func TestValidateToken(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	tests := []struct {
		name      string
		setupFunc func() string
		wantErr   error
	}{
		{
			name: "valid token",
			setupFunc: func() string {
				token, _ := manager.GenerateToken(userID)
				return token
			},
			wantErr: nil,
		},
		{
			name: "empty token",
			setupFunc: func() string {
				return ""
			},
			wantErr: ErrMissingToken,
		},
		{
			name: "expired token",
			setupFunc: func() string {
				// Create a manager with very short expiry
				shortManager := NewJWTManager(privateKey, publicKey, 1*time.Millisecond)
				token, _ := shortManager.GenerateToken(userID)
				time.Sleep(2 * time.Millisecond)
				return token
			},
			wantErr: ErrTokenExpired,
		},
		{
			name: "invalid signature",
			setupFunc: func() string {
				// Generate token with different key
				otherPrivateKey, _ := generateTestKeyPair(t)
				otherManager := NewJWTManager(otherPrivateKey, publicKey, 60*time.Minute)
				token, _ := otherManager.GenerateToken(userID)
				return token
			},
			wantErr: ErrInvalidSignature,
		},
		{
			name: "malformed token",
			setupFunc: func() string {
				return "not.a.valid.jwt.token"
			},
			wantErr: nil, // Will be wrapped in "failed to parse token"
		},
		{
			name: "invalid issuer",
			setupFunc: func() string {
				// Create token with different issuer
				config := JWTManagerConfig{
					PrivateKey: privateKey,
					PublicKey:  publicKey,
					Expiry:     60 * time.Minute,
					Issuer:     "wrong-issuer",
					Audience:   "cairn-api",
				}
				wrongManager := NewJWTManagerWithConfig(config)
				token, _ := wrongManager.GenerateToken(userID)
				return token
			},
			wantErr: ErrInvalidIssuer,
		},
		{
			name: "invalid audience",
			setupFunc: func() string {
				// Create token with different audience
				config := JWTManagerConfig{
					PrivateKey: privateKey,
					PublicKey:  publicKey,
					Expiry:     60 * time.Minute,
					Issuer:     "cairn-user-service",
					Audience:   "wrong-audience",
				}
				wrongManager := NewJWTManagerWithConfig(config)
				token, _ := wrongManager.GenerateToken(userID)
				return token
			},
			wantErr: ErrInvalidAudience,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupFunc()
			claims, err := manager.ValidateToken(token)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, claims)
			} else if tt.name == "valid token" {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, userID, claims.UserID)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestParseTokenWithoutValidation(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 1*time.Millisecond)
	userID := uuid.New()

	// Generate an expired token
	token, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(2 * time.Millisecond)

	// Validation should fail
	_, err = manager.ValidateToken(token)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenExpired)

	// Parse without validation should succeed
	claims, err := manager.ParseTokenWithoutValidation(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   error
	}{
		{
			name:      "valid bearer token",
			header:    "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:   nil,
		},
		{
			name:      "valid bearer token with lowercase",
			header:    "bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:   nil,
		},
		{
			name:      "empty header",
			header:    "",
			wantToken: "",
			wantErr:   ErrMissingToken,
		},
		{
			name:      "missing bearer scheme",
			header:    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "",
			wantErr:   ErrInvalidAuthHeader,
		},
		{
			name:      "wrong scheme",
			header:    "Basic eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "",
			wantErr:   ErrInvalidAuthHeader,
		},
		{
			name:      "bearer without token",
			header:    "Bearer ",
			wantToken: "",
			wantErr:   ErrMissingToken,
		},
		{
			name:      "bearer with empty token",
			header:    "Bearer",
			wantToken: "",
			wantErr:   ErrInvalidAuthHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractTokenFromHeader(tt.header)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}

func TestGetTokenExpiry(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	expiry := 45 * time.Minute
	manager := NewJWTManager(privateKey, publicKey, expiry)

	assert.Equal(t, expiry, manager.GetTokenExpiry())
}

func TestUpdateKeys(t *testing.T) {
	oldPrivateKey, oldPublicKey := generateTestKeyPair(t)
	newPrivateKey, newPublicKey := generateTestKeyPair(t)

	manager := NewJWTManager(oldPrivateKey, oldPublicKey, 60*time.Minute)
	userID := uuid.New()

	// Generate token with old keys
	oldToken, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Validate with old keys
	claims, err := manager.ValidateToken(oldToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)

	// Update to new keys
	manager.UpdateKeys(newPrivateKey, newPublicKey)

	// Old token should fail validation with new keys
	_, err = manager.ValidateToken(oldToken)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSignature)

	// Generate new token with new keys
	newToken, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// New token should validate with new keys
	claims, err = manager.ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestGetPublicKey(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)

	retrievedKey := manager.GetPublicKey()
	assert.Equal(t, publicKey, retrievedKey)
}

func TestGetTokenInfo(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	info, err := manager.GetTokenInfo(token)
	require.NoError(t, err)

	assert.Equal(t, userID, info.UserID)
	assert.Equal(t, "cairn-user-service", info.Issuer)
	assert.Contains(t, info.Audience, "cairn-api")
	assert.Equal(t, userID.String(), info.Subject)
	assert.False(t, info.IsExpired)
	assert.Greater(t, info.TimeLeft, time.Duration(0))
	assert.LessOrEqual(t, info.TimeLeft, 60*time.Minute)
	assert.WithinDuration(t, time.Now(), info.IssuedAt, 5*time.Second)
	assert.WithinDuration(t, time.Now().Add(60*time.Minute), info.ExpiresAt, 5*time.Second)
}

func TestGetTokenInfo_ExpiredToken(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 1*time.Millisecond)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(2 * time.Millisecond)

	info, err := manager.GetTokenInfo(token)
	require.NoError(t, err)

	assert.True(t, info.IsExpired)
	assert.Equal(t, time.Duration(0), info.TimeLeft)
}

func TestTokenClaims(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Parse token manually to verify all claims
	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	require.NoError(t, err)

	claims, ok := parsedToken.Claims.(*Claims)
	require.True(t, ok)

	// Verify all standard claims
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "cairn-user-service", claims.Issuer)
	assert.Contains(t, claims.Audience, "cairn-api")
	assert.Equal(t, userID.String(), claims.Subject)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.NotBefore)

	// Verify timing
	now := time.Now()
	assert.WithinDuration(t, now, claims.IssuedAt.Time, 5*time.Second)
	assert.WithinDuration(t, now.Add(60*time.Minute), claims.ExpiresAt.Time, 5*time.Second)
	assert.WithinDuration(t, now, claims.NotBefore.Time, 5*time.Second)
}

func TestRS256SigningMethod(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Parse token to check signing method
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	require.NoError(t, err)

	assert.Equal(t, "RS256", parsedToken.Method.Alg())
}

func TestValidateToken_NotBeforeCheck(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	// Create token with nbf in the future
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cairn-user-service",
			Audience:  jwt.ClaimStrings{"cairn-api"},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(60 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(10 * time.Second)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err)

	// Validation should fail because token is not yet valid
	_, err = manager.ValidateToken(tokenString)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkGenerateToken(b *testing.B) {
	privateKey, publicKey := generateTestKeyPair(&testing.T{})
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateToken(userID)
	}
}

func BenchmarkValidateToken(b *testing.B) {
	privateKey, publicKey := generateTestKeyPair(&testing.T{})
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()
	token, _ := manager.GenerateToken(userID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateToken(token)
	}
}

// Tests for kid (Key ID) functionality

func TestComputeKeyID(t *testing.T) {
	_, publicKey := generateTestKeyPair(t)

	t.Run("generates consistent key ID", func(t *testing.T) {
		keyID1, err := ComputeKeyID(publicKey)
		require.NoError(t, err)

		keyID2, err := ComputeKeyID(publicKey)
		require.NoError(t, err)

		assert.Equal(t, keyID1, keyID2, "Key ID should be consistent")
		assert.Len(t, keyID1, 16, "Key ID should be 16 characters")
	})

	t.Run("different keys have different IDs", func(t *testing.T) {
		_, publicKey2 := generateTestKeyPair(t)

		keyID1, err := ComputeKeyID(publicKey)
		require.NoError(t, err)

		keyID2, err := ComputeKeyID(publicKey2)
		require.NoError(t, err)

		assert.NotEqual(t, keyID1, keyID2, "Different keys should have different IDs")
	})

	t.Run("nil key returns error", func(t *testing.T) {
		_, err := ComputeKeyID(nil)
		assert.Error(t, err, "Expected error for nil key")
	})
}

func TestNewJWTManager_ComputesKeyID(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)

	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)

	expectedKeyID, _ := ComputeKeyID(publicKey)
	assert.Equal(t, expectedKeyID, manager.GetKeyID(), "Key ID should be computed from public key")
	assert.NotEmpty(t, manager.GetKeyID(), "Key ID should not be empty")
}

func TestNewJWTManagerWithConfig_KeyID(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)

	t.Run("uses provided key ID", func(t *testing.T) {
		config := JWTManagerConfig{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			Expiry:     60 * time.Minute,
			KeyID:      "custom-key-id",
		}

		manager := NewJWTManagerWithConfig(config)
		assert.Equal(t, "custom-key-id", manager.GetKeyID())
	})

	t.Run("computes key ID when not provided", func(t *testing.T) {
		config := JWTManagerConfig{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			Expiry:     60 * time.Minute,
		}

		manager := NewJWTManagerWithConfig(config)

		expectedKeyID, _ := ComputeKeyID(publicKey)
		assert.Equal(t, expectedKeyID, manager.GetKeyID())
	})
}

func TestUpdateKeys_UpdatesKeyID(t *testing.T) {
	oldPrivateKey, oldPublicKey := generateTestKeyPair(t)
	newPrivateKey, newPublicKey := generateTestKeyPair(t)

	manager := NewJWTManager(oldPrivateKey, oldPublicKey, 60*time.Minute)
	originalKeyID := manager.GetKeyID()

	manager.UpdateKeys(newPrivateKey, newPublicKey)

	newKeyID := manager.GetKeyID()
	expectedKeyID, _ := ComputeKeyID(newPublicKey)

	assert.Equal(t, expectedKeyID, newKeyID, "Key ID should be updated to match new key")
	assert.NotEqual(t, originalKeyID, newKeyID, "Key ID should change after key update")
}

func TestGetKeyID(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)

	keyID := manager.GetKeyID()

	assert.NotEmpty(t, keyID, "Key ID should not be empty")

	expectedKeyID, _ := ComputeKeyID(publicKey)
	assert.Equal(t, expectedKeyID, keyID)
}

func TestGenerateToken_IncludesKidHeader(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Parse the token to check the kid header
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	require.NoError(t, err)

	// Verify kid header is present and matches expected value
	kid, ok := parsedToken.Header["kid"].(string)
	assert.True(t, ok, "kid header should be present")

	expectedKeyID := manager.GetKeyID()
	assert.Equal(t, expectedKeyID, kid, "kid header should match manager's key ID")
}

func TestGenerateToken_KidHeaderChangesWithKeyUpdate(t *testing.T) {
	oldPrivateKey, oldPublicKey := generateTestKeyPair(t)
	newPrivateKey, newPublicKey := generateTestKeyPair(t)

	manager := NewJWTManager(oldPrivateKey, oldPublicKey, 60*time.Minute)
	userID := uuid.New()

	// Generate token with old keys
	oldToken, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Parse to get old kid
	oldParsedToken, err := jwt.Parse(oldToken, func(token *jwt.Token) (interface{}, error) {
		return oldPublicKey, nil
	})
	require.NoError(t, err)
	oldKid := oldParsedToken.Header["kid"].(string)

	// Update keys
	manager.UpdateKeys(newPrivateKey, newPublicKey)

	// Generate token with new keys
	newToken, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// Parse to get new kid
	newParsedToken, err := jwt.Parse(newToken, func(token *jwt.Token) (interface{}, error) {
		return newPublicKey, nil
	})
	require.NoError(t, err)
	newKid := newParsedToken.Header["kid"].(string)

	// Kids should be different after key update
	assert.NotEqual(t, oldKid, newKid, "kid header should change after key update")
}
