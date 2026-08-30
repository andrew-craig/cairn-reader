package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	pkgauth "github.com/andrew-craig/cairn-reader/pkg/auth"
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

	// Verify the minted token validates through the canonical pkg/auth validator
	// (the real request-path validator; JWTManager no longer validates).
	validator := pkgauth.NewValidator(publicKey)
	claims, err := validator.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "cairn-user-service", claims.Issuer)
	assert.Contains(t, claims.Audience, "cairn-api")
	assert.Equal(t, userID.String(), claims.Subject)

	// jti must be present and unique per token - RS256 signing is
	// deterministic, so without a jti two tokens minted for the same user
	// within the same second would be byte-identical.
	assert.NotEmpty(t, claims.ID)

	token2, err := manager.GenerateToken(userID)
	require.NoError(t, err)
	claims2, err := validator.ValidateToken(token2)
	require.NoError(t, err)
	assert.NotEmpty(t, claims2.ID)
	assert.NotEqual(t, claims.ID, claims2.ID)
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

	// Validate with old keys (via the canonical pkg/auth validator)
	claims, err := pkgauth.NewValidator(oldPublicKey).ValidateToken(oldToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)

	// Update to new keys
	manager.UpdateKeys(newPrivateKey, newPublicKey)

	// Old token should fail validation against the new key
	_, err = pkgauth.NewValidator(newPublicKey).ValidateToken(oldToken)
	assert.ErrorIs(t, err, pkgauth.ErrInvalidSignature)

	// Generate new token with new keys
	newToken, err := manager.GenerateToken(userID)
	require.NoError(t, err)

	// New token should validate with new keys
	claims, err = pkgauth.NewValidator(newPublicKey).ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestGetPublicKey(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)

	retrievedKey := manager.GetPublicKey()
	assert.Equal(t, publicKey, retrievedKey)
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

// TestUpdateKeys_ConcurrentAccess verifies thread-safety of key updates.
// This test checks that concurrent token generation and key updates don't
// cause race conditions or mismatched key usage.
func TestUpdateKeys_ConcurrentAccess(t *testing.T) {
	privateKey1, publicKey1 := generateTestKeyPair(t)
	privateKey2, publicKey2 := generateTestKeyPair(t)

	manager := NewJWTManager(privateKey1, publicKey1, 60*time.Minute)
	userID := uuid.New()

	// Run with race detector: go test -race
	const numGoroutines = 100
	done := make(chan bool, numGoroutines*2)

	// Start goroutines that generate tokens
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			token, err := manager.GenerateToken(userID)
			if err != nil {
				t.Errorf("GenerateToken failed: %v", err)
				return
			}
			// Token should be non-empty
			if token == "" {
				t.Error("Generated empty token")
			}
		}()
	}

	// Start goroutines that update keys
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer func() { done <- true }()
			// Alternate between key pairs
			if i%2 == 0 {
				manager.UpdateKeys(privateKey1, publicKey1)
			} else {
				manager.UpdateKeys(privateKey2, publicKey2)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}

// TestConcurrentGenerateAndValidate tests that tokens can be generated and
// validated concurrently without race conditions.
func TestConcurrentGenerateAndValidate(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	manager := NewJWTManager(privateKey, publicKey, 60*time.Minute)

	validator := pkgauth.NewValidator(publicKey)

	const numGoroutines = 50
	tokens := make(chan string, numGoroutines)
	done := make(chan bool, numGoroutines*2)

	// Start goroutines that generate tokens
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			userID := uuid.New()
			token, err := manager.GenerateToken(userID)
			if err != nil {
				t.Errorf("GenerateToken failed: %v", err)
				return
			}
			tokens <- token
		}()
	}

	// Start goroutines that validate tokens
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			select {
			case token := <-tokens:
				claims, err := validator.ValidateToken(token)
				if err != nil {
					t.Errorf("ValidateToken failed: %v", err)
					return
				}
				if claims == nil {
					t.Error("ValidateToken returned nil claims")
				}
			case <-time.After(100 * time.Millisecond):
				// Timeout waiting for token, ok to skip
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}

// TestConcurrentGetPublicKeyAndUpdate tests that GetPublicKey and UpdateKeys
// can be called concurrently without race conditions.
func TestConcurrentGetPublicKeyAndUpdate(t *testing.T) {
	privateKey1, publicKey1 := generateTestKeyPair(t)
	privateKey2, publicKey2 := generateTestKeyPair(t)

	manager := NewJWTManager(privateKey1, publicKey1, 60*time.Minute)

	const numGoroutines = 100
	done := make(chan bool, numGoroutines*2)

	// Start goroutines that get public key
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			key := manager.GetPublicKey()
			if key == nil {
				t.Error("GetPublicKey returned nil")
			}
		}()
	}

	// Start goroutines that update keys
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer func() { done <- true }()
			if i%2 == 0 {
				manager.UpdateKeys(privateKey1, publicKey1)
			} else {
				manager.UpdateKeys(privateKey2, publicKey2)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}

// TestConcurrentGetKeyIDAndUpdate tests that GetKeyID and UpdateKeys
// can be called concurrently without race conditions.
func TestConcurrentGetKeyIDAndUpdate(t *testing.T) {
	privateKey1, publicKey1 := generateTestKeyPair(t)
	privateKey2, publicKey2 := generateTestKeyPair(t)

	manager := NewJWTManager(privateKey1, publicKey1, 60*time.Minute)

	const numGoroutines = 100
	done := make(chan bool, numGoroutines*2)

	// Start goroutines that get key ID
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			keyID := manager.GetKeyID()
			if keyID == "" {
				t.Error("GetKeyID returned empty string")
			}
		}()
	}

	// Start goroutines that update keys
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer func() { done <- true }()
			if i%2 == 0 {
				manager.UpdateKeys(privateKey1, publicKey1)
			} else {
				manager.UpdateKeys(privateKey2, publicKey2)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}
