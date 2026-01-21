package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	pkgapi "github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/api/dto"
	"github.com/go-chi/chi/v5"
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
// POST /api/v1/explore/article
func (s *Server) handleArticles(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, maxArticlesBatchSize)

	var payload dto.ArticlesRequest

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

	// Validate the request
	if err := payload.Validate(); err != nil {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeValidation, err.Error(), nil, "v1")
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
// GET /api/v1/explore/recommendation/{user_id}
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from JWT token context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
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
	// Extract authenticated user ID from JWT token context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()

	// Extract article ID from chi URL parameter
	articleID := chi.URLParam(r, "article_id")

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
	// Extract authenticated user ID from JWT token context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()

	// Extract article ID from chi URL parameter
	articleID := chi.URLParam(r, "article_id")

	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, maxSimpleRequestSize)

	var payload dto.VoteRequest

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

	// Validate the request
	if err := payload.Validate(); err != nil {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeValidation, err.Error(), nil, "v1")
		return
	}

	if err := s.voteRepo.RecordVote(r.Context(), userID, articleID, payload.VoteType); err != nil {
		slog.Error("failed to record vote", slog.Any("error", err))

		// Check for specific errors to return appropriate status codes
		if errors.Is(err, apperrors.ErrArticleNotFound) {
			pkgapi.WriteError(w, http.StatusNotFound, pkgapi.ErrCodeNotFound, "Article not found", nil, "v1")
			return
		}
		if errors.Is(err, apperrors.ErrInvalidVoteType) {
			pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeValidation, "Invalid vote type", nil, "v1")
			return
		}
		if errors.Is(err, apperrors.ErrInvalidUserID) {
			pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeValidation, "Invalid user ID", nil, "v1")
			return
		}

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
	// Extract authenticated user ID from JWT token context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()

	// Extract article ID from chi URL parameter
	articleID := chi.URLParam(r, "article_id")

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
	// Extract article ID from chi URL parameter
	articleID := chi.URLParam(r, "article_id")

	upvotes, downvotes, err := s.voteRepo.GetVoteCounts(r.Context(), articleID)
	if err != nil {
		slog.Error("failed to get vote counts", slog.Any("error", err))

		// Check for specific errors to return appropriate status codes
		if errors.Is(err, apperrors.ErrArticleNotFound) {
			pkgapi.WriteError(w, http.StatusNotFound, pkgapi.ErrCodeNotFound, "Article not found", nil, "v1")
			return
		}

		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to get vote counts", nil, "v1")
		return
	}

	// Get the authenticated user's vote using the JWT context
	// TODO: Consider removing the query parameter option and always using JWT user ID
	userVote := ""
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
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
