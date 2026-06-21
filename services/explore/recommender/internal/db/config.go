package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func envInt32(key string, fallback int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
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

// Connect establishes a database connection pool
func (c *Config) Connect(ctx context.Context) (*pgxpool.Pool, error) {
	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "require"
	}
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
	config.MaxConnLifetime = envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	config.MaxConnIdleTime = envDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)

	// Statement timeout: prevent slow queries from exhausting the pool.
	// Overridable via DB_STATEMENT_TIMEOUT (e.g. "30s", "60s").
	if timeout := envDuration("DB_STATEMENT_TIMEOUT", 30*time.Second); timeout > 0 {
		if config.ConnConfig.RuntimeParams == nil {
			config.ConnConfig.RuntimeParams = make(map[string]string)
		}
		config.ConnConfig.RuntimeParams["statement_timeout"] = fmt.Sprintf("%d", timeout.Milliseconds())
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("connected to recommender database",
		slog.String("host", c.Host),
		slog.String("port", c.Port),
	)
	return pool, nil
}
