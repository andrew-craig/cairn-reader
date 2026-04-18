package db

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

func envInt32(key string, fallback int32) int32 {
	if v := env.GetString(key, ""); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return fallback
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// NewConfigFromEnv creates a Config from environment variables
func NewConfigFromEnv() *Config {
	return &Config{
		Host:     env.GetString("DB_HOST", "localhost"),
		Port:     env.GetString("DB_PORT", "5432"),
		User:     env.GetString("DB_USER", "fetcher"),
		Password: env.GetString("DB_PASSWORD", "fetcher_password"),
		DBName:   env.GetString("DB_NAME", "fetcher_db"),
	}
}

// Connect establishes a database connection pool
func (c *Config) Connect(ctx context.Context) (*pgxpool.Pool, error) {
	sslMode := env.GetString("DB_SSLMODE", "require")
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, sslMode,
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Connection pool settings. Overridable via DB_MAX_CONNS / DB_MIN_CONNS
	// so constrained deployments (e.g. selfhost) can cap pool size below the
	// server's max_connections.
	config.MaxConns = envInt32("DB_MAX_CONNS", 25)
	config.MinConns = envInt32("DB_MIN_CONNS", 5)
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("connected to fetcher database",
		slog.String("host", c.Host),
		slog.String("port", c.Port),
	)
	return pool, nil
}
