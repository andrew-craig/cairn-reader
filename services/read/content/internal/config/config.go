// Package config provides configuration management for the content service.
// It loads and validates configuration from environment variables and provides
// structured access to service settings.
package config

import (
	"fmt"

	sharedconfig "github.com/cairn-app/cairn-reader/pkg/config"
)

// VaultConfig contains HashiCorp Vault configuration
type VaultConfig struct {
	Address       string
	Token         string // For token-based auth (development)
	RoleID        string // For AppRole authentication (production)
	SecretID      string // For AppRole authentication (production)
	AuthPath      string // AppRole auth path (default: "approle")
	PublicKeyPath string
}

// HasAppRoleAuth returns true if AppRole credentials are configured
func (c *VaultConfig) HasAppRoleAuth() bool {
	return c.RoleID != "" && c.SecretID != ""
}

// HasTokenAuth returns true if token-based auth is configured
func (c *VaultConfig) HasTokenAuth() bool {
	return c.Token != ""
}

// Config holds all configuration for the content service
type Config struct {
	Server              sharedconfig.ServerConfig
	Database            sharedconfig.DatabaseConfig
	Logging             sharedconfig.LoggingConfig
	Vault               VaultConfig
	IngestRSSServiceURL   string
	EmailIngestServiceURL string
	InternalAPIKey        string
}

// Load reads configuration from environment variables and validates it
func Load() (*Config, error) {
	cfg := &Config{
		Server:   sharedconfig.LoadServerConfig("8080"),
		Database: sharedconfig.LoadDatabaseConfig("cairn_content", "cairn_content_pass", "cairn_content"),
		Logging:  sharedconfig.LoadLoggingConfig(),
		Vault: VaultConfig{
			Address:       sharedconfig.GetString("VAULT_ADDR", "http://localhost:8200"),
			Token:         sharedconfig.GetString("VAULT_TOKEN", ""),
			RoleID:        sharedconfig.GetString("VAULT_ROLE_ID", ""),
			SecretID:      sharedconfig.GetString("VAULT_SECRET_ID", ""),
			AuthPath:      sharedconfig.GetString("VAULT_AUTH_PATH", "approle"),
			PublicKeyPath: sharedconfig.GetString("JWT_PUBLIC_KEY_PATH", "secret/data/jwt/public-key"),
		},
		IngestRSSServiceURL:   sharedconfig.GetString("INGEST_RSS_SERVICE_URL", "http://localhost:8085"),
		EmailIngestServiceURL: sharedconfig.GetString("EMAIL_INGEST_SERVICE_URL", "http://localhost:8087"),
		InternalAPIKey:      sharedconfig.GetString("INTERNAL_API_KEY", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}

	// Validate Vault configuration
	if c.Vault.Address == "" {
		return fmt.Errorf("VAULT_ADDR is required")
	}

	// Require either token auth or AppRole auth
	if !c.Vault.HasTokenAuth() && !c.Vault.HasAppRoleAuth() {
		return fmt.Errorf("VAULT_TOKEN or VAULT_ROLE_ID/VAULT_SECRET_ID is required")
	}

	if c.Vault.PublicKeyPath == "" {
		return fmt.Errorf("JWT_PUBLIC_KEY_PATH is required")
	}

	if c.InternalAPIKey == "" {
		return fmt.Errorf("INTERNAL_API_KEY is required for service-to-service authentication")
	}

	return nil
}
