package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Common JWT validation errors
var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token has expired")
	ErrInvalidSignature  = errors.New("invalid token signature")
	ErrInvalidIssuer     = errors.New("invalid token issuer")
	ErrInvalidAudience   = errors.New("invalid token audience")
	ErrMissingToken      = errors.New("missing authorization token")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// Validator handles JWT token validation using RS256 public key
type Validator struct {
	publicKey *rsa.PublicKey
	keyID     string // Key ID (kid) for identifying which key to use for validation
	issuer    string
	audience  string
}

// ComputeKeyID generates a key ID from an RSA public key.
// The key ID is the first 16 characters of the base64url-encoded SHA256 hash
// of the DER-encoded public key. This provides a stable, unique identifier
// that can be used to select the correct key during validation.
func ComputeKeyID(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", errors.New("public key cannot be nil")
	}

	// Encode public key to DER format
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(derBytes)

	// Base64url encode and truncate to 16 characters for readability
	// 16 chars = 96 bits of entropy, sufficient for key identification
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])
	if len(encoded) > 16 {
		encoded = encoded[:16]
	}

	return encoded, nil
}

// ValidatorConfig holds configuration for the JWT validator
type ValidatorConfig struct {
	PublicKey *rsa.PublicKey
	KeyID     string // Key ID (kid) - if empty, computed from public key
	Issuer    string // Expected issuer claim (optional, validates if set)
	Audience  string // Expected audience claim (optional, validates if set)
}

// NewValidator creates a new JWT validator with a public key
func NewValidator(publicKey *rsa.PublicKey) *Validator {
	// Compute key ID from public key (ignore error, use empty string if computation fails)
	keyID, _ := ComputeKeyID(publicKey)

	return &Validator{
		publicKey: publicKey,
		keyID:     keyID,
		issuer:    "cairn-user-service",
		audience:  "cairn-api",
	}
}

// NewValidatorWithConfig creates a new JWT validator with custom configuration
func NewValidatorWithConfig(config ValidatorConfig) *Validator {
	// Use provided key ID or compute from public key
	keyID := config.KeyID
	if keyID == "" && config.PublicKey != nil {
		keyID, _ = ComputeKeyID(config.PublicKey)
	}

	v := &Validator{
		publicKey: config.PublicKey,
		keyID:     keyID,
		issuer:    config.Issuer,
		audience:  config.Audience,
	}

	// Set defaults if not provided
	if v.issuer == "" {
		v.issuer = "cairn-user-service"
	}
	if v.audience == "" {
		v.audience = "cairn-api"
	}

	return v
}

// ValidateToken validates a JWT token and returns the claims
func (v *Validator) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Check kid (Key ID) header if present
		// This enables key rotation by allowing validators to identify which key to use
		if tokenKid, ok := token.Header["kid"].(string); ok && tokenKid != "" {
			if v.keyID != "" && tokenKid != v.keyID {
				// Log warning but continue validation - the signature check will determine validity
				// In a multi-key setup, this is where you'd look up the correct key by kid
				slog.Warn("JWT kid mismatch - token may have been signed with a different key",
					"token_kid", tokenKid,
					"expected_kid", v.keyID,
				)
			}
		}

		return v.publicKey, nil
	})

	if err != nil {
		// Check for specific error types
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate issuer if configured
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, ErrInvalidIssuer
	}

	// Validate audience if configured
	if v.audience != "" {
		validAudience := false
		for _, aud := range claims.Audience {
			if aud == v.audience {
				validAudience = true
				break
			}
		}
		if !validAudience {
			return nil, ErrInvalidAudience
		}
	}

	return claims, nil
}

// ExtractTokenFromHeader extracts the JWT token from an Authorization header
// Expected format: "Bearer <token>"
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrMissingToken
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", ErrInvalidAuthHeader
	}

	scheme := parts[0]
	token := parts[1]

	if !strings.EqualFold(scheme, "Bearer") {
		return "", ErrInvalidAuthHeader
	}

	if token == "" {
		return "", ErrMissingToken
	}

	return token, nil
}

// UpdatePublicKey updates the RSA public key (useful for key rotation)
// Note: This also recomputes the key ID based on the new public key
func (v *Validator) UpdatePublicKey(publicKey *rsa.PublicKey) {
	v.publicKey = publicKey
	// Recompute key ID for the new key
	v.keyID, _ = ComputeKeyID(publicKey)
}

// GetPublicKey returns the current public key
func (v *Validator) GetPublicKey() *rsa.PublicKey {
	return v.publicKey
}

// GetKeyID returns the current key ID (kid)
func (v *Validator) GetKeyID() string {
	return v.keyID
}

// TokenInfo holds information about a token
type TokenInfo struct {
	UserID    uuid.UUID
	IssuedAt  time.Time
	ExpiresAt time.Time
	NotBefore time.Time
	Issuer    string
	Audience  []string
	Subject   string
	IsExpired bool
	TimeLeft  time.Duration
}

// ParseTokenWithoutValidation parses a token without validating the signature
// WARNING: This should only be used for debugging or extracting claims from expired tokens
// NEVER use this for authorization decisions
func (v *Validator) ParseTokenWithoutValidation(tokenString string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetTokenInfo extracts and returns information about a token without full validation
// Useful for debugging and logging purposes
func (v *Validator) GetTokenInfo(tokenString string) (*TokenInfo, error) {
	claims, err := v.ParseTokenWithoutValidation(tokenString)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := claims.ExpiresAt.Time
	isExpired := now.After(expiresAt)
	timeLeft := time.Duration(0)
	if !isExpired {
		timeLeft = time.Until(expiresAt)
	}

	return &TokenInfo{
		UserID:    claims.UserID,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiresAt: expiresAt,
		NotBefore: claims.NotBefore.Time,
		Issuer:    claims.Issuer,
		Audience:  claims.Audience,
		Subject:   claims.Subject,
		IsExpired: isExpired,
		TimeLeft:  timeLeft,
	}, nil
}
