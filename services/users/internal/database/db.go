// Package database provides PostgreSQL database connectivity and repository
// implementations for the user service. It uses pgx for high-performance
// PostgreSQL operations with connection pooling.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgx connection pool and provides health checking capabilities.
// It should be created via New() and closed via Close() when the application shuts down.
type DB struct {
	Pool *pgxpool.Pool
}

// Config holds PostgreSQL database connection configuration.
// All fields except SSLMode are required for a successful connection.
type Config struct {
	Host             string        // Database server hostname or IP address
	Port             string        // Database server port (typically "5432")
	User             string        // Database username
	Password         string        // Database password
	Database         string        // Database name to connect to
	SSLMode          string        // SSL mode: "disable", "require", "verify-ca", "verify-full"
	MaxOpenConns     int32         // Maximum number of open connections in the pool
	MaxIdleConns     int32         // Minimum number of idle connections maintained in the pool
	ConnMaxLifetime  time.Duration // Maximum lifetime of a connection before it's closed and replaced
	StatementTimeout time.Duration // Per-connection statement timeout (0 = no timeout)
}

// New creates a new database connection
func New(cfg *Config) (*DB, error) {
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	config.MaxConns = cfg.MaxOpenConns
	config.MinConns = cfg.MaxIdleConns
	config.MaxConnLifetime = cfg.ConnMaxLifetime

	// Statement timeout: prevent slow queries from exhausting the pool.
	if cfg.StatementTimeout > 0 {
		if config.ConnConfig.RuntimeParams == nil {
			config.ConnConfig.RuntimeParams = make(map[string]string)
		}
		config.ConnConfig.RuntimeParams["statement_timeout"] = fmt.Sprintf("%d", cfg.StatementTimeout.Milliseconds())
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Health checks the database connection health
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
