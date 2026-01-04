package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	pkgapi "github.com/andrew-craig/cairn/pkg/api"
	"github.com/andrew-craig/cairn/pkg/auth"
	"github.com/andrew-craig/cairn/pkg/models"
)

const (
	// Request body size limits
	maxArticlesBatchSize = 10 << 20 // 10MB for batch article submission
	maxSimpleRequestSize = 1 << 10  // 1KB for simple JSON requests
)

// handleLiveness returns the liveness status (liveness probe)
// This endpoint indicates if the process is running
// Used by orchestrators to determine if the service should be restarted
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	}); err != nil {
		slog.Error("failed to encode liveness response", slog.Any("error", err))
	}
}

// handleReadiness returns the readiness status (readiness probe)
// This endpoint checks if dependencies (database, vault) are available
// Returns 503 Service Unavailable if dependencies are unreachable
// Used by load balancers to determine if traffic should be routed to this instance
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity with 5s timeout
	ctx, cancel := r.Context(), func() {}
	if _, hasDeadline := r.Context().Deadline(); !hasDeadline {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithTimeout(r.Context(), 5*time.Second)
		cancel = cancelFunc
	}
	defer cancel()

	checks := make(map[string]string)
	status := "healthy"
	statusCode := http.StatusOK

	// Check database connectivity
	if err := s.db.Ping(ctx); err != nil {
		checks["database"] = "error"
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
		slog.Warn("database health check failed", slog.Any("error", err))
	} else {
		checks["database"] = "ok"
	}

	// TODO: Add Vault connectivity check if needed
	// For now, we only check database as Vault is only used at startup

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"checks": checks,
	}); err != nil {
		slog.Error("failed to encode readiness response", slog.Any("error", err))
	}
}

// handleArticles receives articles from the fetcher
func (s *Server) handleArticles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
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
			pkgapi.WriteError(w, http.StatusRequestEntityTooLarge, pkgapi.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	if len(payload.Articles) == 0 {
		pkgapi.WriteSuccess(w, http.StatusOK, map[string]string{"status": "no articles to process"}, "v1")
		return
	}

	// Store articles in database
	if err := s.articleRepo.CreateBatch(r.Context(), payload.Articles); err != nil {
		slog.Error("failed to store articles", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to store articles", nil, "v1")
		return
	}

	slog.Info("successfully stored articles", slog.Int("count", len(payload.Articles)))

	pkgapi.WriteSuccess(w, http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"count":   len(payload.Articles),
		"message": "Articles stored successfully",
	}, "v1")
}

// handleRecommendations returns recommended articles for a user
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	// Get recommendations
	recommendations, err := s.engine.GetRecommendations(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get recommendations", slog.String("user_id", userID), slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to get recommendations", nil, "v1")
		return
	}

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]interface{}{
		"user_id":         userID,
		"recommendations": recommendations,
		"count":           len(recommendations),
	}, "v1")
}

// handleMarkAsRead marks an article as read for a user
// POST /api/v1/explore/article/{article_id}/read
func (s *Server) handleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	// Extract article ID from path: /api/v1/explore/article/{article_id}/read
	articleID := extractPathParam(r.URL.Path, "/api/v1/explore/article/", "/read")

	if articleID == "" {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeBadRequest, "article_id is required", nil, "v1")
		return
	}

	if err := s.userRepo.MarkArticleAsRead(r.Context(), userID, articleID); err != nil {
		slog.Error("failed to mark article as read", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to mark article as read", nil, "v1")
		return
	}

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Article marked as read",
	}, "v1")
}

// handleVote handles upvoting or downvoting an article
// POST /api/v1/explore/article/{article_id}/vote
func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	// Extract article ID from path: /api/v1/explore/article/{article_id}/vote
	articleID := extractPathParam(r.URL.Path, "/api/v1/explore/article/", "/vote")

	if articleID == "" {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeBadRequest, "Article ID is required", nil, "v1")
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
			pkgapi.WriteError(w, http.StatusRequestEntityTooLarge, pkgapi.ErrCodeBadRequest, "Request body too large", nil, "v1")
			return
		}
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeBadRequest, "Invalid request body", nil, "v1")
		return
	}

	if payload.VoteType != "upvote" && payload.VoteType != "downvote" {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeValidation, "vote_type must be 'upvote' or 'downvote'", nil, "v1")
		return
	}

	if err := s.voteRepo.RecordVote(r.Context(), userID, articleID, payload.VoteType); err != nil {
		slog.Error("failed to record vote", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to record vote", nil, "v1")
		return
	}

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Vote recorded successfully",
	}, "v1")
}

// handleRemoveVote removes a user's vote from an article
// DELETE /api/v1/explore/article/{article_id}/vote
func (s *Server) handleRemoveVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
		return
	}

	// Extract authenticated user ID from JWT token context
	authenticatedUserID := auth.MustGetUserID(r.Context())
	userID := authenticatedUserID.String()

	// Extract article ID from path: /api/v1/explore/article/{article_id}/vote
	articleID := extractPathParam(r.URL.Path, "/api/v1/explore/article/", "/vote")

	if articleID == "" {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeBadRequest, "Article ID is required", nil, "v1")
		return
	}

	if err := s.voteRepo.RemoveVote(r.Context(), userID, articleID); err != nil {
		slog.Error("failed to remove vote", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to remove vote", nil, "v1")
		return
	}

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Vote removed successfully",
	}, "v1")
}

// handleGetVotes returns vote counts for an article
// GET /api/v1/explore/article/{article_id}/vote
func (s *Server) handleGetVotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		pkgapi.WriteError(w, http.StatusMethodNotAllowed, pkgapi.ErrCodeMethodNotAllowed, "Method not allowed", nil, "v1")
		return
	}

	// Extract article ID from path: /api/v1/explore/article/{article_id}/vote
	articleID := extractPathParam(r.URL.Path, "/api/v1/explore/article/", "/vote")

	if articleID == "" {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeBadRequest, "Article ID is required", nil, "v1")
		return
	}

	upvotes, downvotes, err := s.voteRepo.GetVoteCounts(r.Context(), articleID)
	if err != nil {
		slog.Error("failed to get vote counts", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to get vote counts", nil, "v1")
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

	pkgapi.WriteSuccess(w, http.StatusOK, response, "v1")
}

// extractPathParam extracts a path parameter from a URL path
// by removing a prefix and suffix, then trimming whitespace.
// Example: extractPathParam("/explore/articles/123/vote", "/explore/articles/", "/vote") returns "123"
func extractPathParam(path, prefix, suffix string) string {
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	return strings.TrimSpace(path)
}
