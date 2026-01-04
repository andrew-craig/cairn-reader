// Package config provides configuration management for the recommender service.
// It loads and validates configuration from environment variables and provides
// structured access to service settings.
package config

import (
	"fmt"

	sharedconfig "github.com/andrew-craig/cairn/pkg/config"
)

// VaultConfig contains HashiCorp Vault configuration
type VaultConfig struct {
	Address        string
	Token          string
	PublicKeyPath  string
}

// Validate checks if the vault configuration is valid
func (c *VaultConfig) Validate() error {
	// Vault is optional for development, but required for production
	// We'll check this based on the environment in the main config validation
	return nil
}

// Config holds all configuration for the recommender service
type Config struct {
	Server             sharedconfig.ServerConfig
	Database           sharedconfig.DatabaseConfig
	Logging            sharedconfig.LoggingConfig
	Vault              VaultConfig
	ArticleRetentionDays int
}

// Load reads configuration from environment variables and validates it
func Load() (*Config, error) {
	cfg := &Config{
		Server:   sharedconfig.LoadServerConfig("8081"),
		Database: sharedconfig.LoadDatabaseConfig("cairn", "cairn_password", "cairn_db"),
		Logging:  sharedconfig.LoadLoggingConfig(),
		Vault: VaultConfig{
			Address:       sharedconfig.GetString("VAULT_ADDR", "http://localhost:8200"),
			Token:         sharedconfig.GetString("VAULT_TOKEN", ""),
			PublicKeyPath: sharedconfig.GetString("JWT_PUBLIC_KEY_PATH", "secret/jwt/public-key"),
		},
		ArticleRetentionDays: sharedconfig.GetInt("ARTICLE_RETENTION_DAYS", 90),
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
		if c.Vault.Token == "" {
			return fmt.Errorf("VAULT_TOKEN is required in production")
		}
	}

	if c.ArticleRetentionDays <= 0 {
		return fmt.Errorf("ARTICLE_RETENTION_DAYS must be greater than 0")
	}

	return nil
}
