package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/andrew-craig/cairn-core/user-service/pkg/auth"
	"github.com/andrew-craig/cairn-explore/pkg/models"
)

// handleHealth returns the health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "recommender",
	}); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

// handleArticles receives articles from the fetcher
func (s *Server) handleArticles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Articles []models.Article `json:"articles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(payload.Articles) == 0 {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "no articles to process"}); err != nil {
			log.Printf("error encoding response: %v", err)
		}
		return
	}

	// Store articles in database
	if err := s.articleRepo.CreateBatch(r.Context(), payload.Articles); err != nil {
		log.Printf("Error storing articles: %v", err)
		http.Error(w, "Failed to store articles", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully stored %d articles", len(payload.Articles))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"count":   len(payload.Articles),
		"message": "Articles stored successfully",
	}); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

// handleRecommendations returns recommended articles for a user
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	// Get recommendations
	recommendations, err := s.engine.GetRecommendations(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting recommendations for user %s: %v", userID, err)
		http.Error(w, "Failed to get recommendations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":         userID,
		"recommendations": recommendations,
		"count":           len(recommendations),
	})
}

// handleMarkAsRead marks an article as read for a user
func (s *Server) handleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	var payload struct {
		ArticleID string `json:"article_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.ArticleID == "" {
		http.Error(w, "article_id is required", http.StatusBadRequest)
		return
	}

	if err := s.userRepo.MarkArticleAsRead(r.Context(), userID, payload.ArticleID); err != nil {
		log.Printf("Error marking article as read: %v", err)
		http.Error(w, "Failed to mark article as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Article marked as read",
	})
}

// handleVote handles upvoting or downvoting an article
// POST /explore/articles/:id/vote
func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	// Extract article ID from path: /explore/articles/{articleID}/vote
	path := strings.TrimPrefix(r.URL.Path, "/explore/articles/")
	path = strings.TrimSuffix(path, "/vote")
	articleID := strings.TrimSpace(path)

	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	var payload struct {
		VoteType string `json:"vote_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.VoteType != "upvote" && payload.VoteType != "downvote" {
		http.Error(w, "vote_type must be 'upvote' or 'downvote'", http.StatusBadRequest)
		return
	}

	if err := s.voteRepo.RecordVote(r.Context(), userID, articleID, payload.VoteType); err != nil {
		log.Printf("Error recording vote: %v", err)
		http.Error(w, "Failed to record vote", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Vote recorded successfully",
	})
}

// handleRemoveVote removes a user's vote from an article
// DELETE /explore/articles/:id/vote
func (s *Server) handleRemoveVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	// Extract article ID from path: /explore/articles/{articleID}/vote
	path := strings.TrimPrefix(r.URL.Path, "/explore/articles/")
	path = strings.TrimSuffix(path, "/vote")
	articleID := strings.TrimSpace(path)

	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	if err := s.voteRepo.RemoveVote(r.Context(), userID, articleID); err != nil {
		log.Printf("Error removing vote: %v", err)
		http.Error(w, "Failed to remove vote", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Vote removed successfully",
	})
}

// handleGetVotes returns vote counts for an article
// GET /explore/articles/:id/votes
func (s *Server) handleGetVotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract article ID from path: /explore/articles/{articleID}/votes
	path := strings.TrimPrefix(r.URL.Path, "/explore/articles/")
	path = strings.TrimSuffix(path, "/votes")
	articleID := strings.TrimSpace(path)

	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	upvotes, downvotes, err := s.voteRepo.GetVoteCounts(r.Context(), articleID)
	if err != nil {
		log.Printf("Error getting vote counts: %v", err)
		http.Error(w, "Failed to get vote counts", http.StatusInternalServerError)
		return
	}

	// Optionally get user's vote if user_id is provided as query parameter
	userVote := ""
	userID := r.URL.Query().Get("user_id")
	if userID != "" {
		userVote, err = s.voteRepo.GetUserVote(r.Context(), userID, articleID)
		if err != nil {
			log.Printf("Error getting user vote: %v", err)
			// Don't fail the request, just log the error
		}
	}

	response := map[string]interface{}{
		"article_id": articleID,
		"upvotes":    upvotes,
		"downvotes":  downvotes,
	}

	if userVote != "" {
		response["user_vote"] = userVote
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
