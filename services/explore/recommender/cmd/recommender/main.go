package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn-core/user-service/pkg/auth"
	"github.com/andrew-craig/cairn-explore/recommender/internal/api"
	"github.com/andrew-craig/cairn-explore/recommender/internal/cleanup"
	"github.com/andrew-craig/cairn-explore/recommender/internal/db"
	"github.com/andrew-craig/cairn-explore/recommender/internal/recommend"
	_ "github.com/lib/pq"
)

func main() {
	port := getEnv("PORT", "8081")

	// Database configuration
	dbConfig := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "cairn"),
		Password: getEnv("DB_PASSWORD", "cairn_password"),
		DBName:   getEnv("DB_NAME", "cairn_db"),
	}

	// Connect to database
	database, err := connectDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize Vault client and fetch JWT public key for authentication
	vaultAddr := getEnv("VAULT_ADDR", "http://localhost:8200")
	vaultToken := getEnv("VAULT_TOKEN", "")
	publicKeyPath := getEnv("JWT_PUBLIC_KEY_PATH", "secret/jwt/public-key")

	log.Printf("Connecting to Vault at %s", vaultAddr)
	vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
		Address: vaultAddr,
		Token:   vaultToken,
	})
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Fetch JWT public key from Vault
	log.Printf("Fetching JWT public key from Vault path: %s", publicKeyPath)
	publicKey, err := vaultClient.GetPublicKey(publicKeyPath)
	if err != nil {
		log.Fatalf("Failed to get JWT public key from Vault: %v", err)
	}

	// Create JWT validator and auth middleware
	validator := auth.NewValidator(publicKey)
	authMiddleware := auth.NewMiddleware(validator)
	log.Println("JWT authentication middleware initialized")

	// Initialize repositories
	articleRepo := db.NewArticleRepository(database)
	userRepo := db.NewUserRepository(database)
	articleRepo.SetUserRepository(userRepo) // Set user repo for user ID conversion
	voteRepo := db.NewVoteRepository(database, userRepo)

	// Initialize recommendation engine
	recommendEngine := recommend.NewEngine(articleRepo, userRepo)

	// Initialize article cleanup job
	retentionDays := getEnvAsInt("ARTICLE_RETENTION_DAYS", 90)
	cleanupInterval := 24 * time.Hour // Run cleanup once per day
	cleanupJob := cleanup.NewArticleCleanup(articleRepo, retentionDays, cleanupInterval)
	cleanupJob.Start()
	log.Printf("Article cleanup job started (retention: %d days, interval: %s)", retentionDays, cleanupInterval)

	// Initialize API server with auth middleware
	server := api.NewServer(articleRepo, userRepo, voteRepo, recommendEngine, authMiddleware)
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down recommender service...")

		// Stop cleanup job
		cleanupJob.Stop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	log.Printf("Recommender service starting on port %s", port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func connectDB(config db.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DBName)

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := database.Ping(); err == nil {
			log.Println("Successfully connected to database")
			return database, nil
		}
		log.Printf("Waiting for database to be ready... (%d/30)", i+1)
		time.Sleep(time.Second)
	}

	return nil, fmt.Errorf("database not ready after 30 seconds")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid value for %s: %s, using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
