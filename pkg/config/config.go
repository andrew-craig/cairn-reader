// Package config provides shared configuration utilities and patterns
// for all Cairn services. It includes helper functions for reading
// environment variables with type conversion and validation support.
//
// This package follows the configuration pattern established in the
// User Service and provides building blocks for service-specific
// configuration management.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host             string
	Port             string
	User             string
	Password         string
	DBName           string
	SSLMode          string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
	StatementTimeout time.Duration // Per-connection statement timeout (0 = no timeout)
}

// Validate checks if the database configuration is valid
func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Port == "" {
		return fmt.Errorf("DB_PORT is required")
	}
	if c.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	return nil
}

// GetConnectionString returns the PostgreSQL connection string
func (c *DatabaseConfig) GetConnectionString() string {
	s := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.DBName,
		c.SSLMode,
	)
	if c.StatementTimeout > 0 {
		s += fmt.Sprintf(" options=-c statement_timeout=%d", c.StatementTimeout.Milliseconds())
	}
	return s
}

// GetPostgresURL returns the PostgreSQL URL connection string
func (c *DatabaseConfig) GetPostgresURL() string {
	s := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
	if c.StatementTimeout > 0 {
		s += fmt.Sprintf("&options=-c%%20statement_timeout%%3D%d", c.StatementTimeout.Milliseconds())
	}
	return s
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port            string
	Environment     string // "development", "staging", "production"
	ShutdownTimeout time.Duration
}

// Validate checks if the server configuration is valid
func (c *ServerConfig) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	return nil
}

// IsProduction returns true if running in production environment
func (c *ServerConfig) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *ServerConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// Validate checks if the logging configuration is valid
func (c *LoggingConfig) Validate() error {
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Level)
	}
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %s (must be json or text)", c.Format)
	}
	return nil
}

// Helper functions to read environment variables

// GetString returns the environment variable value or default
func GetString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetInt returns the environment variable as int or default
func GetInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetBool returns the environment variable as bool or default
func GetBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetDuration returns the environment variable as duration or default
// Duration can be specified as a time.Duration string (e.g., "30s", "5m", "1h")
// or as a number of seconds
func GetDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Try parsing as duration string first (e.g., "30s", "5m")
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Fall back to parsing as seconds (for backward compatibility)
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

// MustGetString returns the value or panics if not set
func MustGetString(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable not set: %s", key))
	}
	return value
}

// LoadDatabaseConfig loads database configuration from environment variables
func LoadDatabaseConfig(userDefault, passwordDefault, dbnameDefault string) DatabaseConfig {
	return DatabaseConfig{
		Host:             GetString("DB_HOST", "localhost"),
		Port:             GetString("DB_PORT", "5432"),
		User:             GetString("DB_USER", userDefault),
		Password:         GetString("DB_PASSWORD", passwordDefault),
		DBName:           GetString("DB_NAME", dbnameDefault),
		SSLMode:          GetString("DB_SSLMODE", "require"),
		MaxOpenConns:     GetInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:     GetInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime:  GetDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		StatementTimeout: GetDuration("DB_STATEMENT_TIMEOUT", 30*time.Second),
	}
}

// LoadServerConfig loads server configuration from environment variables
func LoadServerConfig(portDefault string) ServerConfig {
	return ServerConfig{
		Port:            GetString("PORT", portDefault),
		Environment:     GetString("ENVIRONMENT", "development"),
		ShutdownTimeout: GetDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
	}
}

// LoadLoggingConfig loads logging configuration from environment variables
func LoadLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:  GetString("LOG_LEVEL", "info"),
		Format: GetString("LOG_FORMAT", "text"),
	}
}
