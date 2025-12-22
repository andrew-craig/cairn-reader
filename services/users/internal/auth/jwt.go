package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
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
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
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
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, expiry time.Duration) *JWTManager {
	return &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		expiry:     expiry,
		issuer:     "cairn-user-service",
		audience:   "cairn-api",
	}
}

// NewJWTManagerWithConfig creates a new JWT manager with custom configuration
func NewJWTManagerWithConfig(config JWTManagerConfig) *JWTManager {
	jm := &JWTManager{
		privateKey: config.PrivateKey,
		publicKey:  config.PublicKey,
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
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(j.privateKey)
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

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.publicKey, nil
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

// UpdateKeys updates the RSA key pair (useful for key rotation)
func (j *JWTManager) UpdateKeys(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) {
	j.privateKey = privateKey
	j.publicKey = publicKey
}

// GetPublicKey returns the current public key (useful for other services)
func (j *JWTManager) GetPublicKey() *rsa.PublicKey {
	return j.publicKey
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
