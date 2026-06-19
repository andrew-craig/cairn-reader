package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
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
	maxShownRequestSize  = 16 << 10 // 16KB for shown batch (up to 100 article IDs)
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

// handleRecommendations returns recommended articles for the authenticated user
// GET /api/v1/explore/recommendation
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from JWT token context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()

	// Parse pagination offset (default 0). Clients advance the offset to page
	// through the ranked feed within a scroll session.
	offset := 0
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsed, err := strconv.Atoi(offsetParam); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get recommendations
	recommendations, err := s.engine.GetRecommendations(r.Context(), userID, offset)
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

// handleMarkShown records that a batch of articles was shown to the
// authenticated user. Clients send this after the articles cross the
// viewport mid-point following a scroll interaction. It is the sole
// writer of `recommendations` rows and the sole driver of the
// `articles.recommends` counter — GetRecommendations is a pure read.
//
// POST /api/v1/explore/shown
func (s *Server) handleMarkShown(w http.ResponseWriter, r *http.Request) {
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()

	r.Body = http.MaxBytesReader(w, r.Body, maxShownRequestSize)

	var payload dto.MarkShownRequest
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

	if err := payload.Validate(); err != nil {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeValidation, err.Error(), nil, "v1")
		return
	}

	if err := s.userRepo.EnsureUserExists(r.Context(), userID); err != nil {
		slog.Error("failed to ensure user exists", slog.String("user_id", userID), slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to record shown articles", nil, "v1")
		return
	}

	recorded := 0
	for _, articleID := range payload.ArticleIDs {
		if ctxErr := r.Context().Err(); ctxErr != nil {
			slog.Warn("context canceled, stopping shown recording",
				slog.String("user_id", userID),
				slog.Any("error", ctxErr))
			break
		}
		inserted, err := s.articleRepo.RecordRecommendation(r.Context(), userID, articleID)
		if err != nil {
			slog.Warn("failed to record shown article",
				slog.String("article_id", articleID),
				slog.String("user_id", userID),
				slog.Any("error", err))
			continue
		}
		if !inserted {
			// Already recorded for this user — don't double-count the
			// articles.recommends counter.
			continue
		}
		if err := s.articleRepo.IncrementRecommendCount(r.Context(), articleID); err != nil {
			if errors.Is(err, apperrors.ErrArticleNotFound) {
				slog.Warn("shown article not found, skipping increment",
					slog.String("article_id", articleID),
					slog.String("user_id", userID))
				continue
			}
			slog.Warn("failed to increment recommend count",
				slog.String("article_id", articleID),
				slog.String("user_id", userID),
				slog.Any("error", err))
			continue
		}
		recorded++
	}

	slog.Info("recorded shown articles",
		slog.String("user_id", userID),
		slog.Int("requested", len(payload.ArticleIDs)),
		slog.Int("recorded", recorded))

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"recorded": recorded,
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
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()
	userVote, err := s.voteRepo.GetUserVote(r.Context(), userID, articleID)
	if err != nil {
		slog.Error("failed to get user vote", slog.Any("error", err))
		// Don't fail the request, just log the error
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

// handleGetUserVoteStats returns aggregate upvote/downvote counts for the authenticated user
// GET /api/v1/explore/user/vote-stats
func (s *Server) handleGetUserVoteStats(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from JWT token context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()

	upvotes, downvotes, err := s.voteRepo.GetUserVoteStats(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get user vote stats", slog.String("user_id", userID), slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to get vote stats", nil, "v1")
		return
	}

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]interface{}{
		"upvotes":   upvotes,
		"downvotes": downvotes,
	}, "v1")
}

// handleSearch searches articles by query string
// GET /api/v1/explore/search?q=...&limit=...&offset=...
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		pkgapi.WriteError(w, http.StatusBadRequest, pkgapi.ErrCodeValidation, "q parameter is required", nil, "v1")
		return
	}

	limit := 20
	offset := 0

	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsed, err := strconv.Atoi(offsetParam); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	articles, err := s.articleRepo.Search(r.Context(), q, limit, offset)
	if err != nil {
		slog.Error("failed to search articles", slog.String("q", q), slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to search articles", nil, "v1")
		return
	}

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]interface{}{
		"articles": articles,
		"count":    len(articles),
		"pagination": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"has_more": len(articles) == limit,
		},
	}, "v1")
}

// handleGetUserVotedArticles returns all articles the authenticated user has voted on
// GET /api/v1/explore/user/votes
func (s *Server) handleGetUserVotedArticles(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from JWT token context
	authenticatedUserID, err := auth.GetUserIDOrError(r.Context())
	if err != nil {
		slog.Error("user ID not found in context", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Authentication context error", nil, "v1")
		return
	}
	userID := authenticatedUserID.String()

	// Parse pagination query params
	limit := 20
	offset := 0

	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsed, err := strconv.Atoi(offsetParam); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	votedArticles, err := s.voteRepo.GetUserVotedArticles(r.Context(), userID, limit, offset)
	if err != nil {
		slog.Error("failed to get user voted articles", slog.Any("error", err))
		pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to get voted articles", nil, "v1")
		return
	}

	pkgapi.WriteSuccess(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"votes":   votedArticles,
		"count":   len(votedArticles),
	}, "v1")
}
