package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// APIKeyRepository defines the interface for API key validation operations
type APIKeyRepository interface {
	// ValidateKey validates an API key and updates last_used_at if valid
	ValidateKey(ctx context.Context, apiKey string) (bool, error)

	// GetKeyHashByName retrieves the key hash for a given key name
	GetKeyHashByName(ctx context.Context, keyName string) (string, error)
}

// apiKeyRepository is the concrete implementation of APIKeyRepository
type apiKeyRepository struct {
	db *sql.DB
}

// NewAPIKeyRepository creates a new APIKeyRepository
func NewAPIKeyRepository(db *sql.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

// ValidateKey validates an API key and updates last_used_at if valid
// Uses constant-time comparison via database to prevent timing attacks
func (r *apiKeyRepository) ValidateKey(ctx context.Context, apiKey string) (bool, error) {
	// Hash the provided API key
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	// Query for active key with this hash and update last_used_at
	query := `
		UPDATE api_keys
		SET last_used_at = $1
		WHERE key_hash = $2
		  AND status = 'active'
		  AND (expires_at IS NULL OR expires_at > NOW())
		RETURNING id
	`

	var id string
	err := r.db.QueryRowContext(ctx, query, time.Now(), keyHash).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil // Key not found or not valid
	}
	if err != nil {
		return false, fmt.Errorf("failed to validate API key: %w", err)
	}

	return true, nil
}

// GetKeyHashByName retrieves the key hash for a given key name
func (r *apiKeyRepository) GetKeyHashByName(ctx context.Context, keyName string) (string, error) {
	query := `
		SELECT key_hash
		FROM api_keys
		WHERE key_name = $1
		  AND status = 'active'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`

	var keyHash string
	err := r.db.QueryRowContext(ctx, query, keyName).Scan(&keyHash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key hash by name: %w", err)
	}

	return keyHash, nil
}
