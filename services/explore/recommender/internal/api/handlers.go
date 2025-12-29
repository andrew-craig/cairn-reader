package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/andrew-craig/cairn/services/users/pkg/auth"
	"github.com/andrew-craig/cairn/services/explore/pkg/models"
)

const (
	// Request body size limits
	maxArticlesBatchSize = 10 << 20 // 10MB for batch article submission
	maxSimpleRequestSize = 1 << 10  // 1KB for simple JSON requests
)

// handleHealth returns the health status (liveness probe)
// This endpoint indicates if the process is running
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "recommender",
	}); err != nil {
		slog.Error("failed to encode health response", slog.Any("error", err))
	}
}

// handleReady returns the readiness status (readiness probe)
// This endpoint checks if dependencies (database) are available
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := s.db.Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"error":  "database unavailable",
		}); err != nil {
			slog.Error("failed to encode readiness response", slog.Any("error", err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "ready",
		"service": "recommender",
	}); err != nil {
		slog.Error("failed to encode readiness response", slog.Any("error", err))
	}
}

// handleArticles receives articles from the fetcher
func (s *Server) handleArticles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, maxArticlesBatchSize)

	var payload struct {
		Articles []models.Article `json:"articles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			slog.Warn("request body too large", slog.Int64("limit", maxBytesErr.Limit))
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(payload.Articles) == 0 {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "no articles to process"}); err != nil {
			slog.Error("failed to encode empty articles response", slog.Any("error", err))
		}
		return
	}

	// Store articles in database
	if err := s.articleRepo.CreateBatch(r.Context(), payload.Articles); err != nil {
		slog.Error("failed to store articles", slog.Any("error", err))
		http.Error(w, "Failed to store articles", http.StatusInternalServerError)
		return
	}

	slog.Info("successfully stored articles", slog.Int("count", len(payload.Articles)))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"count":   len(payload.Articles),
		"message": "Articles stored successfully",
	}); err != nil {
		slog.Error("failed to encode articles response", slog.Any("error", err))
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
		slog.Error("failed to get recommendations", slog.String("user_id", userID), slog.Any("error", err))
		http.Error(w, "Failed to get recommendations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":         userID,
		"recommendations": recommendations,
		"count":           len(recommendations),
	}); err != nil {
		slog.Error("failed to encode recommendations response", slog.Any("error", err))
	}
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

	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, maxSimpleRequestSize)

	var payload struct {
		ArticleID string `json:"article_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			slog.Warn("request body too large", slog.Int64("limit", maxBytesErr.Limit))
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.ArticleID == "" {
		http.Error(w, "article_id is required", http.StatusBadRequest)
		return
	}

	if err := s.userRepo.MarkArticleAsRead(r.Context(), userID, payload.ArticleID); err != nil {
		slog.Error("failed to mark article as read", slog.Any("error", err))
		http.Error(w, "Failed to mark article as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Article marked as read",
	}); err != nil {
		slog.Error("failed to encode mark-as-read response", slog.Any("error", err))
	}
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
	articleID := extractPathParam(r.URL.Path, "/explore/articles/", "/vote")

	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, maxSimpleRequestSize)

	var payload struct {
		VoteType string `json:"vote_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			slog.Warn("request body too large", slog.Int64("limit", maxBytesErr.Limit))
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.VoteType != "upvote" && payload.VoteType != "downvote" {
		http.Error(w, "vote_type must be 'upvote' or 'downvote'", http.StatusBadRequest)
		return
	}

	if err := s.voteRepo.RecordVote(r.Context(), userID, articleID, payload.VoteType); err != nil {
		slog.Error("failed to record vote", slog.Any("error", err))
		http.Error(w, "Failed to record vote", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Vote recorded successfully",
	}); err != nil {
		slog.Error("failed to encode vote response", slog.Any("error", err))
	}
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
	articleID := extractPathParam(r.URL.Path, "/explore/articles/", "/vote")

	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	if err := s.voteRepo.RemoveVote(r.Context(), userID, articleID); err != nil {
		slog.Error("failed to remove vote", slog.Any("error", err))
		http.Error(w, "Failed to remove vote", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Vote removed successfully",
	}); err != nil {
		slog.Error("failed to encode remove-vote response", slog.Any("error", err))
	}
}

// handleGetVotes returns vote counts for an article
// GET /explore/articles/:id/votes
//
// NOTE: This endpoint is protected by auth middleware. The user_id query parameter
// allows fetching the authenticated user's vote status. Consider using the JWT user ID
// directly instead of the query parameter for better security.
func (s *Server) handleGetVotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract article ID from path: /explore/articles/{articleID}/votes
	articleID := extractPathParam(r.URL.Path, "/explore/articles/", "/votes")

	if articleID == "" {
		http.Error(w, "Article ID is required", http.StatusBadRequest)
		return
	}

	upvotes, downvotes, err := s.voteRepo.GetVoteCounts(r.Context(), articleID)
	if err != nil {
		slog.Error("failed to get vote counts", slog.Any("error", err))
		http.Error(w, "Failed to get vote counts", http.StatusInternalServerError)
		return
	}

	// Get the authenticated user's vote using the JWT context
	// TODO: Consider removing the query parameter option and always using JWT user ID
	userVote := ""
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()
	if userID != "" {
		userVote, err = s.voteRepo.GetUserVote(r.Context(), userID, articleID)
		if err != nil {
			slog.Error("failed to get user vote", slog.Any("error", err))
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode votes response", slog.Any("error", err))
	}
}

// extractPathParam extracts a path parameter from a URL path
// by removing a prefix and suffix, then trimming whitespace.
// Example: extractPathParam("/explore/articles/123/vote", "/explore/articles/", "/vote") returns "123"
func extractPathParam(path, prefix, suffix string) string {
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	return strings.TrimSpace(path)
}
