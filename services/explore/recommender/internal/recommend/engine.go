// Package recommend implements the article recommendation algorithm.
// It selects articles for users based on quality scores and exposure metrics.
package recommend

import (
	"context"
	"log/slog"
	"math"
	"sort"

	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
)

// Engine handles recommendation logic
type Engine struct {
	articleRepo db.ArticleRepositoryInterface
	userRepo    db.UserRepositoryInterface
}

// NewEngine creates a new recommendation engine
func NewEngine(articleRepo db.ArticleRepositoryInterface, userRepo db.UserRepositoryInterface) *Engine {
	return &Engine{
		articleRepo: articleRepo,
		userRepo:    userRepo,
	}
}

// recommendationPageSize is the number of articles returned per recommendation
// request. The client paginates through the ranked feed via the offset param.
const recommendationPageSize = 10

// recommendationPoolSize is how many eligible articles are ranked before a page
// is sliced out. The offset window is served from within this pool.
const recommendationPoolSize = 100

// GetRecommendations returns a page of recommended articles for a user, ranked
// by quality score. Callers paginate within a scroll session via offset; the
// page size is fixed at recommendationPageSize.
//
// Quality score formula: (upvotes - (downvotes * 3)) / recommends. Articles
// with recommends == 0 score highest, so low-exposure/new content naturally
// surfaces near the top of the ranking.
//
// This is a pure read: tracking (articles.recommends + recommendations row) is
// driven by POST /api/v1/explore/shown from the client, not by this path.
func (e *Engine) GetRecommendations(ctx context.Context, userID string, offset int) ([]models.Article, error) {
	slog.Info("starting recommendation generation", slog.String("user_id", userID), slog.Int("offset", offset))

	// Ensure user exists
	if err := e.userRepo.EnsureUserExists(ctx, userID); err != nil {
		slog.Error("failed to ensure user exists", slog.String("user_id", userID), slog.Any("error", err))
		return nil, err
	}

	// Get articles eligible for recommendation (not deleted, not already
	// recommended to this user). The pool is over-fetched so the offset window
	// has data to slice from.
	articles, err := e.articleRepo.GetForRecommendation(ctx, userID, recommendationPoolSize)
	if err != nil {
		slog.Error("failed to get eligible articles", slog.String("user_id", userID), slog.Any("error", err))
		return nil, err
	}

	slog.Info("found eligible articles", slog.Int("count", len(articles)), slog.String("user_id", userID))

	// Rank the whole pool by quality, then slice out the requested page.
	e.sortByQuality(articles)

	if offset < 0 {
		offset = 0
	}
	if offset >= len(articles) {
		return []models.Article{}, nil
	}
	end := offset + recommendationPageSize
	if end > len(articles) {
		end = len(articles)
	}
	page := articles[offset:end]

	slog.Info("returning recommendation page",
		slog.Int("offset", offset),
		slog.Int("count", len(page)),
		slog.String("user_id", userID))

	return page, nil
}

// sortByQuality sorts articles in place by descending quality score. Ties are
// broken by article ID (descending) so the ordering is deterministic across
// calls, which keeps offset-based pagination consistent.
func (e *Engine) sortByQuality(articles []models.Article) {
	sort.Slice(articles, func(i, j int) bool {
		si := e.calculateQualityScore(articles[i])
		sj := e.calculateQualityScore(articles[j])
		if si != sj {
			return si > sj
		}
		return articles[i].ID > articles[j].ID
	})
}

// calculateQualityScore computes the quality score for an article
// Formula: (upvotes - (downvotes * 3)) / recommends
// Articles with 0 recommends are treated as having infinite/very high score
func (e *Engine) calculateQualityScore(article models.Article) float64 {
	// Handle division by zero: articles with 0 recommends get very high score
	// This ensures new articles get surfaced quickly
	if article.Recommends == 0 {
		// Return a very high score based on upvotes only
		// This gives new articles with upvotes a chance to be recommended
		if article.Upvotes > 0 {
			return math.Inf(1) // Positive infinity
		}
		// New articles with no votes get a high default score
		return 1000.0
	}

	// Calculate quality score: (upvotes - (downvotes * 3)) / recommends
	// Downvotes heavily penalize quality (3x multiplier)
	// Higher downvotes result in a lower (or negative) score
	numerator := float64(article.Upvotes - (article.Downvotes * 3))
	score := numerator / float64(article.Recommends)

	return score
}
