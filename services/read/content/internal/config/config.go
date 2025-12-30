// Package config provides configuration management for the content service.
// It loads and validates configuration from environment variables and provides
// structured access to service settings.
package config

import (
	"fmt"

	sharedconfig "github.com/andrew-craig/cairn/pkg/config"
)

// Config holds all configuration for the content service
type Config struct {
	Server              sharedconfig.ServerConfig
	Database            sharedconfig.DatabaseConfig
	Logging             sharedconfig.LoggingConfig
	IngestRSSServiceURL string
}

// Load reads configuration from environment variables and validates it
func Load() (*Config, error) {
	cfg := &Config{
		Server:              sharedconfig.LoadServerConfig("8080"),
		Database:            sharedconfig.LoadDatabaseConfig("cairn_content", "cairn_content_pass", "cairn_content"),
		Logging:             sharedconfig.LoadLoggingConfig(),
		IngestRSSServiceURL: sharedconfig.GetString("INGEST_RSS_SERVICE_URL", "http://localhost:8085"),
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

	return nil
}
