// Package config provides configuration management for the fetcher service.
// It loads and validates configuration from environment variables and provides
// structured access to service settings.
package config

import (
	"fmt"
	"time"

	sharedconfig "github.com/cairn-app/cairn-reader/pkg/config"
)

// Config holds all configuration for the fetcher service
type Config struct {
	Server         sharedconfig.ServerConfig
	Database       sharedconfig.DatabaseConfig
	Logging        sharedconfig.LoggingConfig
	RecommenderURL string
	FetchInterval  time.Duration
	KagiFeedURL    string
}

// Load reads configuration from environment variables and validates it
func Load() (*Config, error) {
	cfg := &Config{
		Server:         sharedconfig.LoadServerConfig("8080"),
		Database:       sharedconfig.LoadDatabaseConfig("fetcher", "fetcher_password", "fetcher_db"),
		Logging:        sharedconfig.LoadLoggingConfig(),
		RecommenderURL: sharedconfig.GetString("RECOMMENDER_URL", "http://localhost:8081"),
		FetchInterval:  sharedconfig.GetDuration("FETCH_INTERVAL", 60*time.Second),
		KagiFeedURL:    sharedconfig.GetString("KAGI_FEED_URL", "https://raw.githubusercontent.com/kagisearch/smallweb/main/smallweb.txt"),
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

	if c.RecommenderURL == "" {
		return fmt.Errorf("RECOMMENDER_URL is required")
	}

	if c.FetchInterval <= 0 {
		return fmt.Errorf("FETCH_INTERVAL must be greater than 0")
	}

	if c.KagiFeedURL == "" {
		return fmt.Errorf("KAGI_FEED_URL is required")
	}

	return nil
}
