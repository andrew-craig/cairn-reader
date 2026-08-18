package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Common JWT errors
var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token has expired")
	ErrInvalidSignature  = errors.New("invalid token signature")
	ErrInvalidIssuer     = errors.New("invalid token issuer")
	ErrInvalidAudience   = errors.New("invalid token audience")
	ErrMissingToken      = errors.New("missing authorization token")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
)

// JWTManager handles JWT token operations
type JWTManager struct {
	mu         sync.RWMutex // Protects key fields during rotation
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string // Key ID (kid) for JWT header - identifies which key signed the token
	expiry     time.Duration
	issuer     string
	audience   string
}

// Claims represents JWT claims
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTManagerConfig holds configuration for JWT manager
type JWTManagerConfig struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Expiry     time.Duration
	Issuer     string // Optional: issuer claim for JWT
	Audience   string // Optional: audience claim for JWT
	KeyID      string // Optional: custom key ID (if empty, computed from public key)
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

// NewJWTManager creates a new JWT manager
func NewJWTManager(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, expiry time.Duration) *JWTManager {
	// Compute key ID from public key (ignore error, use empty string if computation fails)
	keyID, _ := ComputeKeyID(publicKey)

	return &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		keyID:      keyID,
		expiry:     expiry,
		issuer:     "cairn-user-service",
		audience:   "cairn-api",
	}
}

// NewJWTManagerWithConfig creates a new JWT manager with custom configuration
func NewJWTManagerWithConfig(config JWTManagerConfig) *JWTManager {
	// Use provided key ID or compute from public key
	keyID := config.KeyID
	if keyID == "" && config.PublicKey != nil {
		keyID, _ = ComputeKeyID(config.PublicKey)
	}

	jm := &JWTManager{
		privateKey: config.PrivateKey,
		publicKey:  config.PublicKey,
		keyID:      keyID,
		expiry:     config.Expiry,
		issuer:     config.Issuer,
		audience:   config.Audience,
	}

	// Set defaults if not provided
	if jm.issuer == "" {
		jm.issuer = "cairn-user-service"
	}
	if jm.audience == "" {
		jm.audience = "cairn-api"
	}

	return jm
}

// GenerateToken generates a new JWT token with standard claims
func (j *JWTManager) GenerateToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Audience:  jwt.ClaimStrings{j.audience},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiry)),
			NotBefore: jwt.NewNumericDate(now),
			// ID (jti) makes every issued token unique. Without it, RS256's
			// deterministic PKCS#1v1.5 padding means two tokens minted for
			// the same user within the same wall-clock second (iat/exp/nbf
			// only have second resolution) are byte-identical, silently
			// defeating rotation.
			ID: uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Hold read lock while accessing key fields
	j.mu.RLock()
	keyID := j.keyID
	privateKey := j.privateKey
	j.mu.RUnlock()

	// Add kid (Key ID) header to identify which key signed the token
	// This is essential for key rotation - validators can use the kid
	// to select the correct public key for signature verification
	if keyID != "" {
		token.Header["kid"] = keyID
	}

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	// Capture public key under read lock for use in the key function
	j.mu.RLock()
	publicKey := j.publicKey
	j.mu.RUnlock()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
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
	if j.issuer != "" && claims.Issuer != j.issuer {
		return nil, ErrInvalidIssuer
	}

	// Validate audience if configured
	if j.audience != "" {
		validAudience := false
		for _, aud := range claims.Audience {
			if aud == j.audience {
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

// ParseTokenWithoutValidation parses a token without validating the signature
// WARNING: This should only be used for debugging or extracting claims from expired tokens
// NEVER use this for authorization decisions
func (j *JWTManager) ParseTokenWithoutValidation(tokenString string) (*Claims, error) {
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

// GetTokenExpiry returns the expiration time for tokens
func (j *JWTManager) GetTokenExpiry() time.Duration {
	return j.expiry
}

// UpdateKeys updates the RSA key pair atomically (useful for key rotation).
// This method is thread-safe and ensures that all key fields are updated
// together, preventing race conditions where token generation could use
// mismatched keys during rotation.
// Note: This also recomputes the key ID based on the new public key
func (j *JWTManager) UpdateKeys(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) {
	// Compute key ID outside the lock since it's a pure function
	keyID, _ := ComputeKeyID(publicKey)

	j.mu.Lock()
	j.privateKey = privateKey
	j.publicKey = publicKey
	j.keyID = keyID
	j.mu.Unlock()
}

// GetPublicKey returns the current public key (useful for other services)
func (j *JWTManager) GetPublicKey() *rsa.PublicKey {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.publicKey
}

// GetKeyID returns the current key ID (kid)
func (j *JWTManager) GetKeyID() string {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.keyID
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

// GetTokenInfo extracts and returns information about a token without full validation
// Useful for debugging and logging purposes
func (j *JWTManager) GetTokenInfo(tokenString string) (*TokenInfo, error) {
	claims, err := j.ParseTokenWithoutValidation(tokenString)
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
