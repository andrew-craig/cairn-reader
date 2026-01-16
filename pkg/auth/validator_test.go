package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Helper function to generate test RSA key pair
func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

// Helper function to generate a test token
func generateTestToken(t *testing.T, privateKey *rsa.PrivateKey, userID uuid.UUID, issuer, audience string, expiry time.Duration) string {
	t.Helper()

	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	return signedToken
}

func TestNewValidator(t *testing.T) {
	_, publicKey := generateTestKeys(t)

	validator := NewValidator(publicKey)

	if validator == nil {
		t.Fatal("Expected validator to be created, got nil")
	}

	if validator.publicKey != publicKey {
		t.Error("Public key not set correctly")
	}

	if validator.issuer != "cairn-user-service" {
		t.Errorf("Expected default issuer 'cairn-user-service', got %s", validator.issuer)
	}

	if validator.audience != "cairn-api" {
		t.Errorf("Expected default audience 'cairn-api', got %s", validator.audience)
	}
}

func TestNewValidatorWithConfig(t *testing.T) {
	_, publicKey := generateTestKeys(t)

	tests := []struct {
		name           string
		config         ValidatorConfig
		expectedIssuer string
		expectedAud    string
	}{
		{
			name: "custom issuer and audience",
			config: ValidatorConfig{
				PublicKey: publicKey,
				Issuer:    "custom-issuer",
				Audience:  "custom-audience",
			},
			expectedIssuer: "custom-issuer",
			expectedAud:    "custom-audience",
		},
		{
			name: "defaults when empty",
			config: ValidatorConfig{
				PublicKey: publicKey,
				Issuer:    "",
				Audience:  "",
			},
			expectedIssuer: "cairn-user-service",
			expectedAud:    "cairn-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidatorWithConfig(tt.config)

			if validator.issuer != tt.expectedIssuer {
				t.Errorf("Expected issuer %s, got %s", tt.expectedIssuer, validator.issuer)
			}

			if validator.audience != tt.expectedAud {
				t.Errorf("Expected audience %s, got %s", tt.expectedAud, validator.audience)
			}
		})
	}
}

func TestValidateToken_Success(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	claims, err := validator.ValidateToken(token)
	if err != nil {
		t.Fatalf("Expected valid token, got error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}

	if claims.Issuer != "cairn-user-service" {
		t.Errorf("Expected issuer 'cairn-user-service', got %s", claims.Issuer)
	}
}

func TestValidateToken_MissingToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)

	_, err := validator.ValidateToken("")
	if err != ErrMissingToken {
		t.Errorf("Expected ErrMissingToken, got %v", err)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	// Create token that expired 1 hour ago
	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", -1*time.Hour)

	_, err := validator.ValidateToken(token)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	privateKey, _ := generateTestKeys(t)
	_, wrongPublicKey := generateTestKeys(t) // Different key pair

	validator := NewValidator(wrongPublicKey)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	_, err := validator.ValidateToken(token)
	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature, got %v", err)
	}
}

func TestValidateToken_InvalidIssuer(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "wrong-issuer", "cairn-api", 1*time.Hour)

	_, err := validator.ValidateToken(token)
	if err != ErrInvalidIssuer {
		t.Errorf("Expected ErrInvalidIssuer, got %v", err)
	}
}

func TestValidateToken_InvalidAudience(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "wrong-audience", 1*time.Hour)

	_, err := validator.ValidateToken(token)
	if err != ErrInvalidAudience {
		t.Errorf("Expected ErrInvalidAudience, got %v", err)
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)

	_, err := validator.ValidateToken("not.a.valid.jwt.token")
	if err == nil {
		t.Error("Expected error for malformed token")
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		expectedErr error
		expectToken bool
	}{
		{
			name:        "valid bearer token",
			header:      "Bearer abc123xyz",
			expectedErr: nil,
			expectToken: true,
		},
		{
			name:        "bearer lowercase",
			header:      "bearer abc123xyz",
			expectedErr: nil,
			expectToken: true,
		},
		{
			name:        "missing token",
			header:      "",
			expectedErr: ErrMissingToken,
			expectToken: false,
		},
		{
			name:        "invalid format",
			header:      "abc123xyz",
			expectedErr: ErrInvalidAuthHeader,
			expectToken: false,
		},
		{
			name:        "wrong scheme",
			header:      "Basic abc123xyz",
			expectedErr: ErrInvalidAuthHeader,
			expectToken: false,
		},
		{
			name:        "empty token",
			header:      "Bearer ",
			expectedErr: ErrMissingToken,
			expectToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractTokenFromHeader(tt.header)

			if err != tt.expectedErr {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}

			if tt.expectToken && token == "" {
				t.Error("Expected token to be extracted")
			}

			if !tt.expectToken && token != "" {
				t.Errorf("Expected no token, got %s", token)
			}
		})
	}
}

func TestUpdatePublicKey(t *testing.T) {
	_, publicKey1 := generateTestKeys(t)
	_, publicKey2 := generateTestKeys(t)

	validator := NewValidator(publicKey1)

	if validator.publicKey != publicKey1 {
		t.Error("Initial public key not set correctly")
	}

	validator.UpdatePublicKey(publicKey2)

	if validator.publicKey != publicKey2 {
		t.Error("Public key not updated correctly")
	}
}

func TestGetPublicKey(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)

	retrievedKey := validator.GetPublicKey()

	if retrievedKey != publicKey {
		t.Error("GetPublicKey did not return the correct key")
	}
}

func TestParseTokenWithoutValidation(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	// Create an expired token
	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", -1*time.Hour)

	// ParseWithoutValidation should succeed even for expired token
	claims, err := validator.ParseTokenWithoutValidation(token)
	if err != nil {
		t.Fatalf("Expected to parse expired token, got error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
}

func TestGetTokenInfo(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	t.Run("valid token", func(t *testing.T) {
		token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

		info, err := validator.GetTokenInfo(token)
		if err != nil {
			t.Fatalf("Expected to get token info, got error: %v", err)
		}

		if info.UserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, info.UserID)
		}

		if info.IsExpired {
			t.Error("Token should not be expired")
		}

		if info.TimeLeft <= 0 {
			t.Error("Time left should be positive")
		}

		if info.Issuer != "cairn-user-service" {
			t.Errorf("Expected issuer 'cairn-user-service', got %s", info.Issuer)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", -1*time.Hour)

		info, err := validator.GetTokenInfo(token)
		if err != nil {
			t.Fatalf("Expected to get token info, got error: %v", err)
		}

		if !info.IsExpired {
			t.Error("Token should be expired")
		}

		if info.TimeLeft != 0 {
			t.Error("Time left should be 0 for expired token")
		}
	})
}

func TestValidateToken_NotBeforeClaim(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	// Create token with NotBefore in the future
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cairn-user-service",
			Audience:  jwt.ClaimStrings{"cairn-api"},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(1 * time.Hour)), // Not valid yet
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	_, err = validator.ValidateToken(signedToken)
	if err == nil {
		t.Error("Expected error for token with future NotBefore claim")
	}
}

func BenchmarkValidateToken(b *testing.B) {
	privateKey, publicKey := generateTestKeys(&testing.T{})
	validator := NewValidator(publicKey)
	userID := uuid.New()
	token := generateTestToken(&testing.T{}, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateToken(token)
	}
}

func BenchmarkExtractTokenFromHeader(b *testing.B) {
	header := "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzNDU2Nzg5MCJ9.signature"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractTokenFromHeader(header)
	}
}

// Tests for kid (Key ID) functionality

func TestComputeKeyID(t *testing.T) {
	_, publicKey := generateTestKeys(t)

	t.Run("generates consistent key ID", func(t *testing.T) {
		keyID1, err := ComputeKeyID(publicKey)
		if err != nil {
			t.Fatalf("Failed to compute key ID: %v", err)
		}

		keyID2, err := ComputeKeyID(publicKey)
		if err != nil {
			t.Fatalf("Failed to compute key ID: %v", err)
		}

		if keyID1 != keyID2 {
			t.Errorf("Key ID should be consistent, got %s and %s", keyID1, keyID2)
		}

		// Key ID should be 16 characters (truncated base64url)
		if len(keyID1) != 16 {
			t.Errorf("Expected key ID length of 16, got %d", len(keyID1))
		}
	})

	t.Run("different keys have different IDs", func(t *testing.T) {
		_, publicKey2 := generateTestKeys(t)

		keyID1, err := ComputeKeyID(publicKey)
		if err != nil {
			t.Fatalf("Failed to compute key ID 1: %v", err)
		}

		keyID2, err := ComputeKeyID(publicKey2)
		if err != nil {
			t.Fatalf("Failed to compute key ID 2: %v", err)
		}

		if keyID1 == keyID2 {
			t.Error("Different keys should have different IDs")
		}
	})

	t.Run("nil key returns error", func(t *testing.T) {
		_, err := ComputeKeyID(nil)
		if err == nil {
			t.Error("Expected error for nil key")
		}
	})
}

func TestNewValidator_ComputesKeyID(t *testing.T) {
	_, publicKey := generateTestKeys(t)

	validator := NewValidator(publicKey)

	expectedKeyID, _ := ComputeKeyID(publicKey)
	if validator.GetKeyID() != expectedKeyID {
		t.Errorf("Expected key ID %s, got %s", expectedKeyID, validator.GetKeyID())
	}
}

func TestNewValidatorWithConfig_KeyID(t *testing.T) {
	_, publicKey := generateTestKeys(t)

	t.Run("uses provided key ID", func(t *testing.T) {
		config := ValidatorConfig{
			PublicKey: publicKey,
			KeyID:     "custom-key-id",
		}

		validator := NewValidatorWithConfig(config)

		if validator.GetKeyID() != "custom-key-id" {
			t.Errorf("Expected key ID 'custom-key-id', got %s", validator.GetKeyID())
		}
	})

	t.Run("computes key ID when not provided", func(t *testing.T) {
		config := ValidatorConfig{
			PublicKey: publicKey,
		}

		validator := NewValidatorWithConfig(config)

		expectedKeyID, _ := ComputeKeyID(publicKey)
		if validator.GetKeyID() != expectedKeyID {
			t.Errorf("Expected computed key ID %s, got %s", expectedKeyID, validator.GetKeyID())
		}
	})
}

func TestUpdatePublicKey_UpdatesKeyID(t *testing.T) {
	_, publicKey1 := generateTestKeys(t)
	_, publicKey2 := generateTestKeys(t)

	validator := NewValidator(publicKey1)
	originalKeyID := validator.GetKeyID()

	validator.UpdatePublicKey(publicKey2)

	newKeyID := validator.GetKeyID()
	expectedKeyID, _ := ComputeKeyID(publicKey2)

	if newKeyID != expectedKeyID {
		t.Errorf("Expected key ID %s after update, got %s", expectedKeyID, newKeyID)
	}

	if newKeyID == originalKeyID {
		t.Error("Key ID should change after updating public key")
	}
}

func TestGetKeyID(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)

	keyID := validator.GetKeyID()

	if keyID == "" {
		t.Error("Expected non-empty key ID")
	}

	expectedKeyID, _ := ComputeKeyID(publicKey)
	if keyID != expectedKeyID {
		t.Errorf("GetKeyID returned %s, expected %s", keyID, expectedKeyID)
	}
}

// Helper function to generate a test token with kid header
func generateTestTokenWithKid(t *testing.T, privateKey *rsa.PrivateKey, userID uuid.UUID, issuer, audience string, expiry time.Duration, kid string) string {
	t.Helper()

	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if kid != "" {
		token.Header["kid"] = kid
	}

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	return signedToken
}

func TestValidateToken_WithMatchingKid(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	expectedKeyID, _ := ComputeKeyID(publicKey)
	token := generateTestTokenWithKid(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour, expectedKeyID)

	claims, err := validator.ValidateToken(token)
	if err != nil {
		t.Fatalf("Expected valid token, got error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
}

func TestValidateToken_WithMismatchedKid(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	// Token with different kid should still validate (signature is correct)
	// but will log a warning
	token := generateTestTokenWithKid(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour, "wrong-kid")

	claims, err := validator.ValidateToken(token)
	if err != nil {
		t.Fatalf("Token should still validate with correct signature, got error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
}

func TestValidateToken_WithoutKid(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)
	userID := uuid.New()

	// Token without kid header should still validate (backward compatibility)
	token := generateTestTokenWithKid(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour, "")

	claims, err := validator.ValidateToken(token)
	if err != nil {
		t.Fatalf("Token without kid should validate, got error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
}

// TestUpdatePublicKey_ConcurrentAccess verifies thread-safety of key updates.
// This test checks that concurrent token validation and key updates don't
// cause race conditions or mismatched key usage.
func TestUpdatePublicKey_ConcurrentAccess(t *testing.T) {
	privateKey1, publicKey1 := generateTestKeys(t)
	_, publicKey2 := generateTestKeys(t)

	validator := NewValidator(publicKey1)
	userID := uuid.New()

	// Generate token with key1
	token := generateTestToken(t, privateKey1, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	// Run with race detector: go test -race
	const numGoroutines = 100
	done := make(chan bool, numGoroutines*2)

	// Start goroutines that validate tokens
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			// Validation may fail if key was updated, but should not panic or race
			_, _ = validator.ValidateToken(token)
		}()
	}

	// Start goroutines that update keys
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer func() { done <- true }()
			// Alternate between key pairs
			if i%2 == 0 {
				validator.UpdatePublicKey(publicKey1)
			} else {
				validator.UpdatePublicKey(publicKey2)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}

// TestConcurrentValidateToken tests that tokens can be validated
// concurrently without race conditions.
func TestConcurrentValidateToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	validator := NewValidator(publicKey)

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	// Start goroutines that validate tokens
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			userID := uuid.New()
			token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

			claims, err := validator.ValidateToken(token)
			if err != nil {
				t.Errorf("ValidateToken failed: %v", err)
				return
			}
			if claims == nil {
				t.Error("ValidateToken returned nil claims")
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestConcurrentGetPublicKeyAndUpdate tests that GetPublicKey and UpdatePublicKey
// can be called concurrently without race conditions.
func TestConcurrentGetPublicKeyAndUpdate(t *testing.T) {
	_, publicKey1 := generateTestKeys(t)
	_, publicKey2 := generateTestKeys(t)

	validator := NewValidator(publicKey1)

	const numGoroutines = 100
	done := make(chan bool, numGoroutines*2)

	// Start goroutines that get public key
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			key := validator.GetPublicKey()
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
				validator.UpdatePublicKey(publicKey1)
			} else {
				validator.UpdatePublicKey(publicKey2)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}

// TestConcurrentGetKeyIDAndUpdate tests that GetKeyID and UpdatePublicKey
// can be called concurrently without race conditions.
func TestConcurrentGetKeyIDAndUpdate(t *testing.T) {
	_, publicKey1 := generateTestKeys(t)
	_, publicKey2 := generateTestKeys(t)

	validator := NewValidator(publicKey1)

	const numGoroutines = 100
	done := make(chan bool, numGoroutines*2)

	// Start goroutines that get key ID
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			keyID := validator.GetKeyID()
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
				validator.UpdatePublicKey(publicKey1)
			} else {
				validator.UpdatePublicKey(publicKey2)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}
