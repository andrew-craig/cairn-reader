package config

import (
	"testing"
	"time"
)

func TestGetConnectionString_StatementTimeout(t *testing.T) {
	cfg := DatabaseConfig{
		Host:             "localhost",
		Port:             "5432",
		User:             "user",
		Password:         "pass",
		DBName:           "db",
		SSLMode:          "disable",
		StatementTimeout: 30 * time.Second,
	}

	s := cfg.GetConnectionString()
	want := "host=localhost port=5432 user=user password=pass dbname=db sslmode=disable options='-c statement_timeout=30000'"
	if s != want {
		t.Errorf("GetConnectionString() = %q, want %q", s, want)
	}
}

func TestGetConnectionString_NoTimeout(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "pass",
		DBName:   "db",
		SSLMode:  "disable",
	}

	s := cfg.GetConnectionString()
	want := "host=localhost port=5432 user=user password=pass dbname=db sslmode=disable"
	if s != want {
		t.Errorf("GetConnectionString() = %q, want %q", s, want)
	}
}

func TestGetPostgresURL_StatementTimeout(t *testing.T) {
	cfg := DatabaseConfig{
		Host:             "localhost",
		Port:             "5432",
		User:             "user",
		Password:         "pass",
		DBName:           "db",
		SSLMode:          "disable",
		StatementTimeout: 30 * time.Second,
	}

	s := cfg.GetPostgresURL()
	want := "postgres://user:pass@localhost:5432/db?sslmode=disable&options=-c%20statement_timeout%3D30000"
	if s != want {
		t.Errorf("GetPostgresURL() = %q, want %q", s, want)
	}
}

func TestGetPostgresURL_NoTimeout(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "pass",
		DBName:   "db",
		SSLMode:  "disable",
	}

	s := cfg.GetPostgresURL()
	want := "postgres://user:pass@localhost:5432/db?sslmode=disable"
	if s != want {
		t.Errorf("GetPostgresURL() = %q, want %q", s, want)
	}
}

func TestLoadDatabaseConfig_StatementTimeoutDefault(t *testing.T) {
	t.Setenv("DB_HOST", "myhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "u")
	t.Setenv("DB_PASSWORD", "p")
	t.Setenv("DB_NAME", "db")

	cfg := LoadDatabaseConfig("u", "p", "db")
	if cfg.StatementTimeout != 30*time.Second {
		t.Errorf("StatementTimeout = %v, want 30s", cfg.StatementTimeout)
	}
}

func TestLoadDatabaseConfig_StatementTimeoutOverride(t *testing.T) {
	t.Setenv("DB_HOST", "myhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "u")
	t.Setenv("DB_PASSWORD", "p")
	t.Setenv("DB_NAME", "db")
	t.Setenv("DB_STATEMENT_TIMEOUT", "10s")

	cfg := LoadDatabaseConfig("u", "p", "db")
	if cfg.StatementTimeout != 10*time.Second {
		t.Errorf("StatementTimeout = %v, want 10s", cfg.StatementTimeout)
	}
}
