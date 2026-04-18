package main

import (
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear all relevant env vars so we exercise the default branch
	envKeys := []string{
		"PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_SSLMODE",
		"DB_NAME_USERS", "DB_NAME_RECOMMENDER", "DB_NAME_FETCHER",
		"DB_NAME_CONTENT", "DB_NAME_RSS", "DB_NAME_EMAIL",
		"JWT_PRIVATE_KEY_FILE", "JWT_PUBLIC_KEY_FILE", "JWT_AUTO_GENERATE",
		"JWT_ACCESS_EXPIRY", "JWT_REFRESH_EXPIRY",
		"INTERNAL_API_KEY", "BCRYPT_COST",
		"KAGI_FEED_URL", "FETCH_INTERVAL",
		"ARTICLE_RETENTION_DAYS", "EMAIL_DOMAIN",
		"LOG_LEVEL", "LOG_FORMAT",
	}
	for _, k := range envKeys {
		t.Setenv(k, "")
	}

	cfg := loadConfig()

	if cfg.Port != "8099" {
		t.Errorf("Port default = %q, want %q", cfg.Port, "8099")
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host default = %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port default = %d, want 5432", cfg.DB.Port)
	}
	if cfg.DBNameUsers != "cairn_users" {
		t.Errorf("DBNameUsers default = %q", cfg.DBNameUsers)
	}
	if cfg.DBNameContent != "content_service" {
		t.Errorf("DBNameContent default = %q", cfg.DBNameContent)
	}
	if cfg.DBNameEmail != "ingest_email" {
		t.Errorf("DBNameEmail default = %q", cfg.DBNameEmail)
	}
	if !cfg.JWTAutoGenerate {
		t.Error("JWTAutoGenerate default should be true")
	}
	if cfg.JWTAccessExpiry != 15*time.Minute {
		t.Errorf("JWTAccessExpiry = %v, want 15m", cfg.JWTAccessExpiry)
	}
	if cfg.JWTRefreshExpiry != 7*24*time.Hour {
		t.Errorf("JWTRefreshExpiry = %v, want 168h", cfg.JWTRefreshExpiry)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost = %d, want 12", cfg.BcryptCost)
	}
	if cfg.ArticleRetentionDays != 90 {
		t.Errorf("ArticleRetentionDays = %d, want 90", cfg.ArticleRetentionDays)
	}
	if cfg.InternalAPIKey == "" {
		t.Error("InternalAPIKey should be auto-generated when unset")
	}
	if len(cfg.InternalAPIKey) < 32 {
		t.Errorf("auto-generated InternalAPIKey too short: %d", len(cfg.InternalAPIKey))
	}
}

func TestLoadConfig_Overrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "override_user")
	t.Setenv("DB_PASSWORD", "override_password")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("DB_NAME_USERS", "my_users")
	t.Setenv("JWT_AUTO_GENERATE", "false")
	t.Setenv("JWT_ACCESS_EXPIRY", "30m")
	t.Setenv("JWT_REFRESH_EXPIRY", "240h")
	t.Setenv("INTERNAL_API_KEY", "fixed-key-value")
	t.Setenv("BCRYPT_COST", "14")
	t.Setenv("ARTICLE_RETENTION_DAYS", "30")
	t.Setenv("EMAIL_DOMAIN", "example.net")
	t.Setenv("LOG_LEVEL", "debug")

	cfg := loadConfig()

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.DB.Host != "db.example.com" {
		t.Errorf("DB.Host = %q", cfg.DB.Host)
	}
	if cfg.DB.Port != 5433 {
		t.Errorf("DB.Port = %d", cfg.DB.Port)
	}
	if cfg.DB.Password != "override_password" {
		t.Errorf("DB.Password = %q", cfg.DB.Password)
	}
	if cfg.DB.SSLMode != "require" {
		t.Errorf("DB.SSLMode = %q", cfg.DB.SSLMode)
	}
	if cfg.DBNameUsers != "my_users" {
		t.Errorf("DBNameUsers = %q", cfg.DBNameUsers)
	}
	if cfg.JWTAutoGenerate {
		t.Error("JWTAutoGenerate should be false after override")
	}
	if cfg.JWTAccessExpiry != 30*time.Minute {
		t.Errorf("JWTAccessExpiry = %v", cfg.JWTAccessExpiry)
	}
	if cfg.JWTRefreshExpiry != 240*time.Hour {
		t.Errorf("JWTRefreshExpiry = %v", cfg.JWTRefreshExpiry)
	}
	if cfg.InternalAPIKey != "fixed-key-value" {
		t.Errorf("InternalAPIKey = %q", cfg.InternalAPIKey)
	}
	if cfg.BcryptCost != 14 {
		t.Errorf("BcryptCost = %d", cfg.BcryptCost)
	}
	if cfg.ArticleRetentionDays != 30 {
		t.Errorf("ArticleRetentionDays = %d", cfg.ArticleRetentionDays)
	}
	if cfg.EmailDomain != "example.net" {
		t.Errorf("EmailDomain = %q", cfg.EmailDomain)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}

func TestLoadConfig_InvalidValuesFallBackToDefaults(t *testing.T) {
	t.Setenv("DB_PORT", "not-a-number")
	t.Setenv("BCRYPT_COST", "nope")
	t.Setenv("JWT_AUTO_GENERATE", "not-a-bool")
	t.Setenv("JWT_ACCESS_EXPIRY", "not-a-duration")

	cfg := loadConfig()

	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want default 5432", cfg.DB.Port)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost = %d, want default 12", cfg.BcryptCost)
	}
	if !cfg.JWTAutoGenerate {
		t.Error("JWTAutoGenerate should fall back to default true")
	}
	if cfg.JWTAccessExpiry != 15*time.Minute {
		t.Errorf("JWTAccessExpiry = %v, want 15m default", cfg.JWTAccessExpiry)
	}
}

func TestDBConfig_ConnString(t *testing.T) {
	db := &DBConfig{
		Host:     "pg",
		Port:     5432,
		User:     "cairn",
		Password: "secret",
		SSLMode:  "disable",
	}
	got := db.ConnString("cairn_users")
	want := "host=pg port=5432 user=cairn password=secret dbname=cairn_users sslmode=disable"
	if got != want {
		t.Errorf("ConnString() =\n  %q\nwant\n  %q", got, want)
	}
}

func TestLoadConfig_InternalAPIKey_Uniqueness(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "")

	cfg1 := loadConfig()
	cfg2 := loadConfig()

	if cfg1.InternalAPIKey == "" || cfg2.InternalAPIKey == "" {
		t.Fatal("auto-generated keys should not be empty")
	}
	if cfg1.InternalAPIKey == cfg2.InternalAPIKey {
		t.Error("auto-generated InternalAPIKey should differ between calls")
	}
}
