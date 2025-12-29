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
