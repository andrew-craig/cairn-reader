// Package main demonstrates integration with the Explore/Recommender service
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/andrew-craig/cairn/services/users/pkg/auth"
)

// RecommendationService handles article recommendations
type RecommendationService struct {
	authMiddleware *auth.Middleware
}

func main() {
	// Get configuration from environment
	vaultAddr := getEnv("VAULT_ADDR", "http://localhost:8200")
	vaultToken := getEnv("VAULT_TOKEN", "dev-token")
	publicKeyPath := getEnv("JWT_PUBLIC_KEY_PATH", "secret/jwt/public-key")
	port := getEnv("PORT", "8081")

	// Setup authentication
	vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
		Address: vaultAddr,
		Token:   vaultToken,
	})
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Fetch JWT public key
	publicKey, err := vaultClient.GetPublicKey(publicKeyPath)
	if err != nil {
		log.Fatalf("Failed to get JWT public key: %v", err)
	}

	// Create validator and middleware
	validator := auth.NewValidator(publicKey)
	middleware := auth.NewMiddleware(validator)

	// Create service
	service := &RecommendationService{
		authMiddleware: middleware,
	}

	// Setup routes
	mux := service.Routes()

	// Start server
	addr := ":" + port
	log.Printf("Recommendation service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// Routes configures all HTTP routes for the service
func (s *RecommendationService) Routes() http.Handler {
	mux := http.NewServeMux()

	// Protected endpoints - require authentication
	mux.Handle("/api/v1/recommendations/", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleGetRecommendations),
	))

	mux.Handle("/api/v1/articles/read", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleMarkArticleRead),
	))

	mux.Handle("/api/v1/articles/vote", s.authMiddleware.RequireAuth(
		http.HandlerFunc(s.handleVoteArticle),
	))

	// Public endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/articles/", s.handleGetArticles)

	return mux
}

// handleGetRecommendations returns personalized article recommendations
func (s *RecommendationService) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from context
	userID := auth.MustGetUserID(r.Context())

	// Optionally validate that path user ID matches authenticated user
	// (implement authorization logic here)

	log.Printf("Getting recommendations for user: %s", userID)

	// Mock recommendations
	response := map[string]interface{}{
		"user_id": userID.String(),
		"recommendations": []map[string]interface{}{
			{
				"article_id": 1,
				"title":      "Understanding Microservices Architecture",
				"score":      0.95,
			},
			{
				"article_id": 2,
				"title":      "JWT Authentication Best Practices",
				"score":      0.89,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMarkArticleRead marks an article as read for the user
func (s *RecommendationService) handleMarkArticleRead(w http.ResponseWriter, r *http.Request) {
	userID := auth.MustGetUserID(r.Context())

	var req struct {
		ArticleID int `json:"article_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("User %s marked article %d as read", userID, req.ArticleID)

	response := map[string]interface{}{
		"success":    true,
		"user_id":    userID.String(),
		"article_id": req.ArticleID,
		"action":     "marked_read",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleVoteArticle handles upvote/downvote on articles
func (s *RecommendationService) handleVoteArticle(w http.ResponseWriter, r *http.Request) {
	userID := auth.MustGetUserID(r.Context())

	var req struct {
		ArticleID int    `json:"article_id"`
		VoteType  string `json:"vote_type"` // "up" or "down"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("User %s voted %s on article %d", userID, req.VoteType, req.ArticleID)

	response := map[string]interface{}{
		"success":    true,
		"user_id":    userID.String(),
		"article_id": req.ArticleID,
		"vote_type":  req.VoteType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetArticles returns a list of articles (public endpoint with optional auth)
func (s *RecommendationService) handleGetArticles(w http.ResponseWriter, r *http.Request) {
	// This endpoint works with or without authentication
	// You could apply OptionalAuth middleware to get user context if available

	response := map[string]interface{}{
		"articles": []map[string]interface{}{
			{
				"id":    1,
				"title": "Understanding Microservices Architecture",
			},
			{
				"id":    2,
				"title": "JWT Authentication Best Practices",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth returns service health status
func (s *RecommendationService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"healthy","service":"recommendation"}`)
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
