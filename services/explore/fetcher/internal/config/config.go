// Package config provides configuration management for the fetcher service.
// It loads and validates configuration from environment variables and provides
// structured access to service settings.
package config

import (
	"fmt"
	"time"

	sharedconfig "github.com/andrew-craig/cairn-reader/pkg/config"
)

// DefaultFeedListURL is the canonical location of the feed list maintained
// in this repository. It's used when no local feed list file is mounted
// into the container.
const DefaultFeedListURL = "https://raw.githubusercontent.com/andrew-craig/cairn-reader/main/services/explore/feeds/default-feeds.txt"

// DefaultFeedListPath is the in-container path the service checks for a
// user-provided feed list. Mount a file here to override the default list.
const DefaultFeedListPath = "/app/feeds/feeds.txt"

// Config holds all configuration for the fetcher service
type Config struct {
	Server         sharedconfig.ServerConfig
	Database       sharedconfig.DatabaseConfig
	Logging        sharedconfig.LoggingConfig
	RecommenderURL string
	FetchInterval  time.Duration
	FeedListPath   string
	FeedListURL    string
	InternalAPIKey string
}

// Load reads configuration from environment variables and validates it
func Load() (*Config, error) {
	cfg := &Config{
		Server:         sharedconfig.LoadServerConfig("8080"),
		Database:       sharedconfig.LoadDatabaseConfig("fetcher", "fetcher_password", "fetcher_db"),
		Logging:        sharedconfig.LoadLoggingConfig(),
		RecommenderURL: sharedconfig.GetString("RECOMMENDER_URL", "http://localhost:8081"),
		FetchInterval:  sharedconfig.GetDuration("FETCH_INTERVAL", 60*time.Second),
		FeedListPath:   sharedconfig.GetString("FEED_LIST_PATH", DefaultFeedListPath),
		FeedListURL:    sharedconfig.GetString("FEED_LIST_URL", DefaultFeedListURL),
		InternalAPIKey: sharedconfig.GetString("INTERNAL_API_KEY", ""),
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

	if c.FeedListPath == "" && c.FeedListURL == "" {
		return fmt.Errorf("one of FEED_LIST_PATH or FEED_LIST_URL must be set")
	}

	if c.InternalAPIKey == "" {
		return fmt.Errorf("INTERNAL_API_KEY is required for service-to-service authentication")
	}

	return nil
}
