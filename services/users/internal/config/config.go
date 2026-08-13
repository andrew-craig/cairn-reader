package config

import (
	"fmt"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/env"
)

// Config holds all configuration for the user service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Vault    VaultConfig
	JWT      JWTConfig
	Security SecurityConfig
	Redis    RedisConfig
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
	Host             string
	Port             string
	User             string
	Password         string
	Database         string
	SSLMode          string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
	StatementTimeout time.Duration // Per-connection statement timeout (0 = no timeout)
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
	PrivateKeyPath      string
	PublicKeyPath       string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
	KeyRotationInterval time.Duration // How often to check for new keys
}

// RedisConfig contains Redis connection configuration for distributed rate limiting
type RedisConfig struct {
	Host     string // Redis host (empty means use in-memory rate limiter)
	Port     string // Redis port (default: 6379)
	Password string // Redis password (optional)
	DB       int    // Redis database index (default: 0)
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	BcryptCost                int
	MinPasswordLength         int
	RequirePasswordComplexity bool
	RateLimitRequests         int
	RateLimitWindow           time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            env.GetString("PORT", "8080"),
			Environment:     env.GetString("ENVIRONMENT", "development"),
			ShutdownTimeout: env.GetDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:             env.GetString("DB_HOST", "localhost"),
			Port:             env.GetString("DB_PORT", "5432"),
			User:             env.GetString("DB_USER", ""),
			Password:         env.GetString("DB_PASSWORD", ""),
			Database:         env.GetString("DB_NAME", "cairn_users"),
			SSLMode:          env.GetString("DB_SSLMODE", "disable"),
			MaxOpenConns:     env.GetInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:     env.GetInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime:  env.GetDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
			StatementTimeout: env.GetDuration("DB_STATEMENT_TIMEOUT", 30*time.Second),
		},
		Vault: VaultConfig{
			Address:              env.GetString("VAULT_ADDR", "http://localhost:8200"),
			Token:                env.GetString("VAULT_TOKEN", ""),
			Namespace:            env.GetString("VAULT_NAMESPACE", ""),
			RoleID:               env.GetString("VAULT_ROLE_ID", ""),
			SecretID:             env.GetString("VAULT_SECRET_ID", ""),
			AuthPath:             env.GetString("VAULT_AUTH_PATH", "approle"),
			DBCredsPath:          env.GetString("VAULT_DB_CREDS_PATH", "secret/data/database/credentials"),
			TokenRenewalInterval: env.GetDuration("VAULT_TOKEN_RENEWAL_INTERVAL", 1*time.Hour),
		},
		JWT: JWTConfig{
			PrivateKeyPath:      env.GetString("JWT_PRIVATE_KEY_PATH", "secret/data/jwt/private-key"),
			PublicKeyPath:       env.GetString("JWT_PUBLIC_KEY_PATH", "secret/data/jwt/public-key"),
			AccessTokenExpiry:   env.GetDuration("JWT_ACCESS_TOKEN_EXPIRY", 60*time.Minute),
			RefreshTokenExpiry:  env.GetDuration("JWT_REFRESH_TOKEN_EXPIRY", 30*24*time.Hour),
			KeyRotationInterval: env.GetDuration("JWT_KEY_ROTATION_INTERVAL", 24*time.Hour),
		},
		Security: SecurityConfig{
			BcryptCost:                env.GetInt("BCRYPT_COST", 12),
			MinPasswordLength:         env.GetInt("MIN_PASSWORD_LENGTH", 8),
			RequirePasswordComplexity: env.GetBool("REQUIRE_PASSWORD_COMPLEXITY", true),
			RateLimitRequests:         env.GetInt("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:           env.GetDuration("RATE_LIMIT_WINDOW", 1*time.Minute),
		},
		Redis: RedisConfig{
			Host:     env.GetString("REDIS_HOST", ""),
			Port:     env.GetString("REDIS_PORT", "6379"),
			Password: env.GetString("REDIS_PASSWORD", ""),
			DB:       env.GetInt("REDIS_DB", 0),
		},
		Logging: LoggingConfig{
			Level:  env.GetString("LOG_LEVEL", "info"),
			Format: env.GetString("LOG_FORMAT", "text"),
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
