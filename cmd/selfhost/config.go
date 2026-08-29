package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the unified configuration for the self-hosted binary.
type Config struct {
	Port string

	// Shared database connection
	DB DBConfig

	// Per-service database names
	DBNameUsers       string
	DBNameRecommender string
	DBNameFetcher     string
	DBNameContent     string
	DBNameRSS         string
	DBNameEmail       string

	// JWT
	JWTPrivateKeyFile string
	JWTPublicKeyFile  string
	JWTAutoGenerate   bool
	JWTAccessExpiry   time.Duration
	JWTRefreshExpiry  time.Duration

	// Auth
	InternalAPIKey    string
	EmailIngestAPIKey string
	BcryptCost        int

	// Explore fetcher
	FeedListPath  string
	FeedListURL   string
	FetchInterval time.Duration

	// Explore recommender
	ArticleRetentionDays int

	// Email
	EmailDomain string

	// Web UI — directory of the bundled static SPA served as a fallback.
	WebDir string

	// Logging
	LogLevel  string
	LogFormat string
}

// DBConfig holds shared database connection parameters.
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	SSLMode  string
}

func (c *DBConfig) ConnString(dbName string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, dbName, c.SSLMode,
	)
}

func loadConfig() *Config {
	internalKey := getEnv("INTERNAL_API_KEY", "")
	if internalKey == "" {
		internalKey = mustGenerateKey()
	}

	emailIngestKey := getEnv("INGEST_API_KEY", "")
	if emailIngestKey == "" {
		emailIngestKey = mustGenerateKey()
	}

	return &Config{
		Port: getEnv("PORT", "8099"),

		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "cairn"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},

		DBNameUsers:       getEnv("DB_NAME_USERS", "cairn_users"),
		DBNameRecommender: getEnv("DB_NAME_RECOMMENDER", "cairn_recommender"),
		DBNameFetcher:     getEnv("DB_NAME_FETCHER", "cairn_fetcher"),
		DBNameContent:     getEnv("DB_NAME_CONTENT", "content_service"),
		DBNameRSS:         getEnv("DB_NAME_RSS", "rss_fetcher_service"),
		DBNameEmail:       getEnv("DB_NAME_EMAIL", "ingest_email"),

		JWTPrivateKeyFile: getEnv("JWT_PRIVATE_KEY_FILE", "/data/keys/private.pem"),
		JWTPublicKeyFile:  getEnv("JWT_PUBLIC_KEY_FILE", "/data/keys/public.pem"),
		JWTAutoGenerate:   getEnvBool("JWT_AUTO_GENERATE", true),
		JWTAccessExpiry:   getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry:  getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),

		InternalAPIKey:    internalKey,
		EmailIngestAPIKey: emailIngestKey,
		BcryptCost:        getEnvInt("BCRYPT_COST", 12),

		FeedListPath:  getEnv("FEED_LIST_PATH", "/data/feeds/feeds.txt"),
		FeedListURL:   getEnv("FEED_LIST_URL", "https://raw.githubusercontent.com/andrew-craig/cairn-reader/main/services/explore/feeds/default-feeds.txt"),
		FetchInterval: getEnvDuration("FETCH_INTERVAL", 60*time.Second),

		ArticleRetentionDays: getEnvInt("ARTICLE_RETENTION_DAYS", 90),

		EmailDomain: getEnv("EMAIL_DOMAIN", "read.cairnapp.com"),

		WebDir: getEnv("WEB_DIR", "/app/web"),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "text"),
	}
}

// mustGenerateKey returns a hex-encoded 32-byte random key, panicking on
// entropy failure. crypto/rand.Read is documented as never failing in
// practice on the platforms we support, so a failure here indicates an
// unrecoverable environment problem and refusing to start is correct.
func mustGenerateKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random key: %v", err))
	}
	return hex.EncodeToString(b)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}

func getEnvBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return b
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultValue
	}
	return d
}
