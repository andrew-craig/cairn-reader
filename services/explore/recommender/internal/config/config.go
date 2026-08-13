// Package config provides configuration management for the recommender service.
// It loads and validates configuration from environment variables and provides
// structured access to service settings.
package config

import (
	"fmt"
	"time"

	sharedconfig "github.com/cairn-app/cairn-reader/pkg/config"
)

// VaultConfig contains HashiCorp Vault configuration
type VaultConfig struct {
	Address                  string
	Token                    string // For token-based auth (development)
	RoleID                   string // For AppRole authentication (production)
	SecretID                 string // For AppRole authentication (production)
	AuthPath                 string // AppRole auth path (default: "approle")
	PublicKeyPath            string
	PublicKeyRefreshInterval time.Duration // How often to re-fetch the JWT public key from Vault; catches key rotation without a restart
}

// Validate checks if the vault configuration is valid
func (c *VaultConfig) Validate() error {
	// Vault is optional for development, but required for production
	// We'll check this based on the environment in the main config validation
	return nil
}

// HasAppRoleAuth returns true if AppRole credentials are configured
func (c *VaultConfig) HasAppRoleAuth() bool {
	return c.RoleID != "" && c.SecretID != ""
}

// HasTokenAuth returns true if token-based auth is configured
func (c *VaultConfig) HasTokenAuth() bool {
	return c.Token != ""
}

// Config holds all configuration for the recommender service
type Config struct {
	Server               sharedconfig.ServerConfig
	Database             sharedconfig.DatabaseConfig
	Logging              sharedconfig.LoggingConfig
	Vault                VaultConfig
	ArticleRetentionDays int
	InternalAPIKey       string
}

// Load reads configuration from environment variables and validates it
func Load() (*Config, error) {
	cfg := &Config{
		Server:   sharedconfig.LoadServerConfig("8081"),
		Database: sharedconfig.LoadDatabaseConfig("cairn", "cairn_password", "cairn_db"),
		Logging:  sharedconfig.LoadLoggingConfig(),
		Vault: VaultConfig{
			Address:                  sharedconfig.GetString("VAULT_ADDR", "http://localhost:8200"),
			Token:                    sharedconfig.GetString("VAULT_TOKEN", ""),
			RoleID:                   sharedconfig.GetString("VAULT_ROLE_ID", ""),
			SecretID:                 sharedconfig.GetString("VAULT_SECRET_ID", ""),
			AuthPath:                 sharedconfig.GetString("VAULT_AUTH_PATH", "approle"),
			PublicKeyPath:            sharedconfig.GetString("JWT_PUBLIC_KEY_PATH", "secret/data/jwt/public-key"),
			PublicKeyRefreshInterval: sharedconfig.GetDuration("JWT_PUBLIC_KEY_REFRESH_INTERVAL", 5*time.Minute),
		},
		ArticleRetentionDays: sharedconfig.GetInt("ARTICLE_RETENTION_DAYS", 90),
		InternalAPIKey:       sharedconfig.GetString("INTERNAL_API_KEY", ""),
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

	// Validate Vault configuration for production
	if c.Server.IsProduction() {
		if c.Vault.Address == "" {
			return fmt.Errorf("VAULT_ADDR is required in production")
		}
		// Require either token auth or AppRole auth
		if !c.Vault.HasTokenAuth() && !c.Vault.HasAppRoleAuth() {
			return fmt.Errorf("VAULT_TOKEN or VAULT_ROLE_ID/VAULT_SECRET_ID is required in production")
		}
	}

	if c.ArticleRetentionDays <= 0 {
		return fmt.Errorf("ARTICLE_RETENTION_DAYS must be greater than 0")
	}

	if c.InternalAPIKey == "" {
		return fmt.Errorf("INTERNAL_API_KEY is required for service-to-service authentication")
	}

	return nil
}
