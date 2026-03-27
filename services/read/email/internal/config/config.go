// Package config provides configuration management for the email ingest service.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds the application configuration
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Email          EmailConfig
	Auth           AuthConfig
	Worker         WorkerConfig
	ContentService ContentServiceConfig
	Logging        LoggingConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port            string
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// EmailConfig holds email-specific settings
type EmailConfig struct {
	Domain string
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	IngestAPIKey      string
	VaultAddr         string
	VaultToken        string
	JWTPublicKeyPath  string
	VaultAuthPath     string
}

// ContentServiceConfig holds configuration for the Content Service HTTP client.
type ContentServiceConfig struct {
	URL     string
	Timeout time.Duration
}

// WorkerConfig holds worker-related settings
type WorkerConfig struct {
	EmailProcessWorkers     int
	EmailProcessPollInterval time.Duration
	OutboxWorkerCount       int
	OutboxPollInterval      time.Duration
	OutboxMaxRetries        int
	RawEmailCleanupCron     string
	RawEmailRetentionDays   int
	OutboxCleanupCron       string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8087"),
			ShutdownTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "cairn_user_email"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "ingest_email"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Email: EmailConfig{
			Domain: getEnv("EMAIL_DOMAIN", "read.cairnapp.com"),
		},
		Auth: AuthConfig{
			IngestAPIKey:     getEnv("INGEST_API_KEY", ""),
			VaultAddr:        getEnv("VAULT_ADDR", "http://localhost:8200"),
			VaultToken:       getEnv("VAULT_TOKEN", ""),
			JWTPublicKeyPath: getEnv("JWT_PUBLIC_KEY_PATH", "secret/data/jwt"),
			VaultAuthPath:    getEnv("VAULT_AUTH_PATH", "approle"),
		},
		Worker: WorkerConfig{
			EmailProcessWorkers:      getEnvInt("EMAIL_PROCESS_WORKERS", 3),
			EmailProcessPollInterval: getEnvDuration("EMAIL_PROCESS_POLL_INTERVAL", 5*time.Second),
			OutboxWorkerCount:        getEnvInt("OUTBOX_WORKER_COUNT", 3),
			OutboxPollInterval:       getEnvDuration("OUTBOX_POLL_INTERVAL", 10*time.Second),
			OutboxMaxRetries:         getEnvInt("OUTBOX_MAX_RETRIES", 6),
			RawEmailCleanupCron:      getEnv("RAW_EMAIL_CLEANUP_CRON", "0 5 * * *"),
			RawEmailRetentionDays:    getEnvInt("RAW_EMAIL_RETENTION_DAYS", 7),
			OutboxCleanupCron:        getEnv("OUTBOX_CLEANUP_CRON", "0 6 * * *"),
		},
		ContentService: ContentServiceConfig{
			URL:     getEnv("CONTENT_SERVICE_URL", "http://localhost:8083"),
			Timeout: getEnvDuration("CONTENT_SERVICE_TIMEOUT", 30*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Validate required fields
	if cfg.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.Auth.IngestAPIKey == "" {
		return nil, fmt.Errorf("INGEST_API_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
