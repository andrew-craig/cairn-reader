// Package config provides configuration management for the RSS fetcher service.
// It loads and validates configuration from environment variables and provides
// structured access to service settings.
package config

import (
	"fmt"

	sharedconfig "github.com/andrew-craig/cairn/pkg/config"
)

// Config holds all configuration for the RSS fetcher service
type Config struct {
	Server   sharedconfig.ServerConfig
	Database sharedconfig.DatabaseConfig
	Logging  sharedconfig.LoggingConfig
}

// Load reads configuration from environment variables and validates it
func Load() (*Config, error) {
	cfg := &Config{
		Server:   sharedconfig.LoadServerConfig("8081"),
		Database: sharedconfig.LoadDatabaseConfig("cairn_rss", "cairn_rss_pass", "cairn_rss"),
		Logging:  sharedconfig.LoadLoggingConfig(),
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
