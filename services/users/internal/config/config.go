package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the user service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Vault    VaultConfig
	JWT      JWTConfig
	Security SecurityConfig
	Logging  LoggingConfig
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port            string
	Environment     string // "development", "staging", "production"
	ShutdownTimeout time.Duration
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// VaultConfig contains HashiCorp Vault configuration
type VaultConfig struct {
	Address              string
	Token                string
	Namespace            string
	RoleID               string // For AppRole authentication
	SecretID             string // For AppRole authentication
	AuthPath             string
	DBCredsPath          string        // Path to database credentials in Vault
	TokenRenewalInterval time.Duration // How often to renew Vault token
}

// JWTConfig contains JWT token configuration
type JWTConfig struct {
	PrivateKeyPath     string
	PublicKeyPath      string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	KeyRotationInterval time.Duration // How often to check for new keys
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	BcryptCost              int
	MinPasswordLength       int
	RequirePasswordComplexity bool
	RateLimitRequests       int
	RateLimitWindow         time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			Environment:     getEnv("ENVIRONMENT", "development"),
			ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			Database:        getEnv("DB_NAME", "cairn_users"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Vault: VaultConfig{
			Address:              getEnv("VAULT_ADDR", "http://localhost:8200"),
			Token:                getEnv("VAULT_TOKEN", ""),
			Namespace:            getEnv("VAULT_NAMESPACE", ""),
			RoleID:               getEnv("VAULT_ROLE_ID", ""),
			SecretID:             getEnv("VAULT_SECRET_ID", ""),
			AuthPath:             getEnv("VAULT_AUTH_PATH", "approle"),
			DBCredsPath:          getEnv("VAULT_DB_CREDS_PATH", "secret/data/database/credentials"),
			TokenRenewalInterval: getDurationEnv("VAULT_TOKEN_RENEWAL_INTERVAL", 1*time.Hour),
		},
		JWT: JWTConfig{
			PrivateKeyPath:      getEnv("JWT_PRIVATE_KEY_PATH", "secret/data/jwt/private-key"),
			PublicKeyPath:       getEnv("JWT_PUBLIC_KEY_PATH", "secret/data/jwt/public-key"),
			AccessTokenExpiry:   getDurationEnv("JWT_ACCESS_TOKEN_EXPIRY", 60*time.Minute),
			RefreshTokenExpiry:  getDurationEnv("JWT_REFRESH_TOKEN_EXPIRY", 30*24*time.Hour),
			KeyRotationInterval: getDurationEnv("JWT_KEY_ROTATION_INTERVAL", 24*time.Hour),
		},
		Security: SecurityConfig{
			BcryptCost:              getIntEnv("BCRYPT_COST", 12),
			MinPasswordLength:       getIntEnv("MIN_PASSWORD_LENGTH", 8),
			RequirePasswordComplexity: getBoolEnv("REQUIRE_PASSWORD_COMPLEXITY", true),
			RateLimitRequests:       getIntEnv("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:         getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate database configuration (only if not using Vault for DB credentials)
	if c.Database.User == "" && c.Vault.Token == "" && c.Vault.RoleID == "" {
		return fmt.Errorf("database user must be set or Vault credentials must be provided")
	}

	// Validate Vault configuration for production
	if c.Server.Environment == "production" {
		if c.Vault.Address == "" {
			return fmt.Errorf("vault address is required in production")
		}
		if c.Vault.Token == "" && (c.Vault.RoleID == "" || c.Vault.SecretID == "") {
			return fmt.Errorf("vault authentication credentials are required in production")
		}
	}

	// Validate security settings
	if c.Security.BcryptCost < 10 || c.Security.BcryptCost > 14 {
		return fmt.Errorf("bcrypt cost must be between 10 and 14")
	}

	if c.Security.MinPasswordLength < 8 {
		return fmt.Errorf("minimum password length must be at least 8 characters")
	}

	return nil
}

// GetDatabaseConnectionString returns the PostgreSQL connection string
func (c *Config) GetDatabaseConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// Helper functions to read environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
