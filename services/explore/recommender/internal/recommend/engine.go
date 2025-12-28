// Package recommend implements the article recommendation algorithm.
// It selects articles for users based on quality scores and exposure metrics.
package recommend

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"

	"github.com/andrew-craig/cairn-explore/pkg/models"
	"github.com/andrew-craig/cairn-explore/recommender/internal/db"
)

// Engine handles recommendation logic
type Engine struct {
	articleRepo *db.ArticleRepository
	userRepo    *db.UserRepository
}

// NewEngine creates a new recommendation engine
func NewEngine(articleRepo *db.ArticleRepository, userRepo *db.UserRepository) *Engine {
	return &Engine{
		articleRepo: articleRepo,
		userRepo:    userRepo,
	}
}

// GetRecommendations returns 5 recommended articles for a user
// Algorithm: 4 high-quality articles (based on quality score) + 1 low-exposure article
// Quality score formula: (upvotes - (downvotes * 3)) / recommends
func (e *Engine) GetRecommendations(ctx context.Context, userID string) ([]models.Article, error) {
	slog.Info("starting recommendation generation", slog.String("user_id", userID))

	// Ensure user exists
	if _, err := e.userRepo.CreateOrGetUser(ctx, userID); err != nil {
		slog.Error("failed to create/get user", slog.String("user_id", userID), slog.Any("error", err))
		return nil, err
	}

	// Get articles eligible for recommendation (not deleted, not already recommended to this user)
	// Request more than needed to allow for quality filtering
	articles, err := e.articleRepo.GetForRecommendation(ctx, userID, 100)
	if err != nil {
		slog.Error("failed to get eligible articles", slog.String("user_id", userID), slog.Any("error", err))
		return nil, err
	}

	slog.Info("found eligible articles", slog.Int("count", len(articles)), slog.String("user_id", userID))

	// If we have 5 or fewer articles, return what we have
	if len(articles) <= 5 {
		// Still need to track recommendations and increment counters
		for _, article := range articles {
			if err := e.trackRecommendation(ctx, userID, article.ID); err != nil {
				slog.Warn("failed to track recommendation", slog.String("article_id", article.ID), slog.Any("error", err))
			}
		}
		return articles, nil
	}

	// Select 4 high-quality articles based on quality score
	highQuality := e.selectHighQualityArticles(articles, 4)

	// Get 1 low-exposure article (lowest recommends count)
	// We'll use a separate query to ensure we get a truly low-exposure article
	lowExposure, err := e.articleRepo.GetLowExposureArticles(ctx, userID, 10)
	if err != nil {
		return nil, err
	}

	// Find a low-exposure article that's not already in high-quality set
	var explorationArticle *models.Article
	highQualityIDs := make(map[string]bool)
	for _, article := range highQuality {
		highQualityIDs[article.ID] = true
	}

	for _, article := range lowExposure {
		if !highQualityIDs[article.ID] {
			explorationArticle = &article
			break
		}
	}

	// Build final recommendation list
	recommended := make([]models.Article, 0, 5)
	recommended = append(recommended, highQuality...)

	// Add exploration article if we found one different from high-quality
	if explorationArticle != nil {
		recommended = append(recommended, *explorationArticle)
	} else if len(highQuality) < 5 && len(lowExposure) > 0 {
		// If we couldn't find a unique exploration article, add more from high quality
		for i := 4; i < len(articles) && len(recommended) < 5; i++ {
			recommended = append(recommended, articles[i])
		}
	}

	// Track recommendations and increment counters
	successCount := 0
	for _, article := range recommended {
		if err := e.trackRecommendation(ctx, userID, article.ID); err != nil {
			slog.Warn("failed to track recommendation",
				slog.String("article_id", article.ID),
				slog.String("user_id", userID),
				slog.Any("error", err))
		} else {
			successCount++
		}
	}

	slog.Info("successfully recommended articles",
		slog.Int("total", len(recommended)),
		slog.String("user_id", userID),
		slog.Int("high_quality", len(highQuality)),
		slog.Int("exploration", len(recommended)-len(highQuality)),
		slog.Int("tracked", successCount))

	// Log article IDs for debugging
	articleIDs := make([]string, len(recommended))
	for i, article := range recommended {
		articleIDs[i] = article.ID
	}
	slog.Debug("recommended article IDs", slog.Any("article_ids", articleIDs))

	return recommended, nil
}

// selectHighQualityArticles selects the top N articles based on quality score.
// It calculates quality scores for all articles and returns the top N by score.
func (e *Engine) selectHighQualityArticles(articles []models.Article, count int) []models.Article {
	type scoredArticle struct {
		article models.Article
		score   float64
	}

	// Calculate quality score for each article
	scored := make([]scoredArticle, len(articles))
	for i, article := range articles {
		score := e.calculateQualityScore(article)
		scored[i] = scoredArticle{article: article, score: score}
	}

	// Sort by score (descending) using sort.Slice (O(n log n))
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Extract top N articles
	result := make([]models.Article, 0, count)
	for i := 0; i < count && i < len(scored); i++ {
		result = append(result, scored[i].article)
	}

	return result
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

// trackRecommendation records the recommendation event and increments counters
func (e *Engine) trackRecommendation(ctx context.Context, userID string, articleID string) error {
	// Increment recommend count
	if err := e.articleRepo.IncrementRecommendCount(ctx, articleID); err != nil {
		return fmt.Errorf("failed to increment recommend count: %w", err)
	}

	// Record in recommendations table
	if err := e.articleRepo.RecordRecommendation(ctx, userID, articleID); err != nil {
		return fmt.Errorf("failed to record recommendation: %w", err)
	}

	slog.Debug("successfully tracked recommendation", slog.String("article_id", articleID), slog.String("user_id", userID))
	return nil
}
