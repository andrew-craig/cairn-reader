// Package fetcher provides RSS feed fetching functionality.
// It fetches feeds from URLs stored in the database, parses RSS/Atom content,
// and submits new articles to the recommender service via HTTP.
package fetcher

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"github.com/andrew-craig/cairn/services/explore/fetcher/internal/client"
	"github.com/andrew-craig/cairn/services/explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn/services/explore/pkg/models"
	"github.com/mmcdole/gofeed"
)

// Fetcher handles RSS feed fetching and article submission.
// It coordinates between the feed repository (for feed management),
// the RSS parser (for parsing feed content), and the recommender client
// (for submitting articles to the recommender service).
type Fetcher struct {
	feedRepo          *db.FeedRepository
	recommenderClient client.RecommenderClientInterface
	parser            *gofeed.Parser
	fetchInterval     time.Duration
}

// NewFetcher creates a new fetcher instance with the specified fetch interval.
// The interval determines how frequently feeds are fetched (e.g., 60s means 1 feed per minute).
func NewFetcher(feedRepo *db.FeedRepository, recommenderClient client.RecommenderClientInterface, fetchInterval time.Duration) *Fetcher {
	return &Fetcher{
		feedRepo:          feedRepo,
		recommenderClient: recommenderClient,
		parser:            gofeed.NewParser(),
		fetchInterval:     fetchInterval,
	}
}

// Run starts the fetch loop that fetches one feed at the configured interval.
// It performs an initial fetch immediately on startup, then continues
// fetching at the configured interval until the context is cancelled.
// The loop is designed to be gentle on feed sources while ensuring
// all feeds are fetched regularly.
func (f *Fetcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	slog.Info("starting fetcher", slog.Duration("interval", f.fetchInterval))

	// Fetch immediately on startup to begin processing feeds right away
	if err := f.FetchSingleFeed(ctx); err != nil {
		slog.Error("initial fetch failed", slog.Any("error", err))
	}

	for {
		select {
		case <-ticker.C:
			if err := f.FetchSingleFeed(ctx); err != nil {
				slog.Error("fetch failed", slog.Any("error", err))
			}
		case <-ctx.Done():
			slog.Info("stopping fetcher")
			return ctx.Err()
		}
	}
}

// FetchSingleFeed fetches one feed and sends articles to the recommender service.
// The fetch process follows these steps:
// 1. Get the next feed to fetch (prioritizing never-fetched, then oldest)
// 2. Fetch and parse the RSS content
// 3. Filter for articles published after the last successful fetch
// 4. Submit new articles to the recommender service
// 5. Record fetch results (success/failure) in the database
func (f *Fetcher) FetchSingleFeed(ctx context.Context) error {
	// Step 1: Get next feed from database (never-fetched feeds have priority)
	feed, err := f.feedRepo.GetNextFeed(ctx)
	if err != nil {
		return fmt.Errorf("get next feed: %w", err)
	}
	if feed == nil {
		return fmt.Errorf("no enabled feeds available")
	}

	slog.Info("fetching feed", slog.Int("feed_id", feed.ID), slog.String("url", feed.URL))

	// Step 2: Fetch and parse RSS/Atom content from the feed URL
	feedData, err := f.fetchRSS(ctx, feed.URL)
	if err != nil {
		if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false); updateErr != nil {
			slog.Error("failed to update fetch result", slog.Int("feed_id", feed.ID), slog.Any("error", updateErr))
		}
		if histErr := f.feedRepo.RecordFetchHistory(ctx, feed.ID, false, 0, 0, err.Error()); histErr != nil {
			slog.Error("failed to record fetch history", slog.Int("feed_id", feed.ID), slog.Any("error", histErr))
		}
		return fmt.Errorf("fetch RSS: %w", err)
	}

	slog.Info("fetched feed", slog.String("title", feedData.Title), slog.Int("items", len(feedData.Items)))

	// Step 3: Filter to only include articles published after last successful fetch
	newArticles := f.filterNewArticles(feedData.Items, feed.LastFetchedAt, feedData, feed.URL)

	slog.Info("found new articles", slog.Int("new_count", len(newArticles)), slog.Int("total_items", len(feedData.Items)))

	// Step 4: Submit new articles to the recommender service via HTTP POST
	articlesSent := 0
	if len(newArticles) > 0 {
		if err := f.recommenderClient.SubmitArticles(ctx, newArticles); err != nil {
			slog.Error("failed to submit articles", slog.Any("error", err))
			if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false); updateErr != nil {
				slog.Error("failed to update fetch result", slog.Int("feed_id", feed.ID), slog.Any("error", updateErr))
			}
			if histErr := f.feedRepo.RecordFetchHistory(ctx, feed.ID, false, len(feedData.Items), 0, err.Error()); histErr != nil {
				slog.Error("failed to record fetch history", slog.Int("feed_id", feed.ID), slog.Any("error", histErr))
			}
			return fmt.Errorf("submit articles: %w", err)
		}
		articlesSent = len(newArticles)
		slog.Info("successfully submitted articles", slog.Int("count", articlesSent))
	}

	// Step 5: Record successful fetch - resets consecutive failure count
	success := true // Success if we completed the fetch, even if no new articles
	if err := f.feedRepo.UpdateFetchResult(ctx, feed.ID, success); err != nil {
		slog.Error("failed to update fetch result", slog.Int("feed_id", feed.ID), slog.Any("error", err))
	}
	if err := f.feedRepo.RecordFetchHistory(ctx, feed.ID, success, len(feedData.Items), articlesSent, ""); err != nil {
		slog.Error("failed to record fetch history", slog.Int("feed_id", feed.ID), slog.Any("error", err))
	}

	return nil
}

// fetchRSS fetches and parses an RSS feed
func (f *Fetcher) fetchRSS(ctx context.Context, feedURL string) (*gofeed.Feed, error) {
	feed, err := f.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	return feed, nil
}

// filterNewArticles returns only articles published after lastFetch.
// For the first fetch of a feed (lastFetch is nil), all articles are returned.
// Uses PublishedParsed as the primary timestamp, falling back to UpdatedParsed
// if PublishedParsed is not available.
func (f *Fetcher) filterNewArticles(items []*gofeed.Item, lastFetch *time.Time, feed *gofeed.Feed, feedURL string) []models.Article {
	if lastFetch == nil {
		// First fetch for this feed - include all articles
		return f.convertItems(items, feed, feedURL)
	}

	var newItems []*gofeed.Item
	for _, item := range items {
		// Prefer PublishedParsed as it represents the original publication date
		if item.PublishedParsed != nil && item.PublishedParsed.After(*lastFetch) {
			newItems = append(newItems, item)
		} else if item.PublishedParsed == nil && item.UpdatedParsed != nil && item.UpdatedParsed.After(*lastFetch) {
			// Fall back to UpdatedParsed only when PublishedParsed is unavailable
			newItems = append(newItems, item)
		}
	}
	return f.convertItems(newItems, feed, feedURL)
}

// convertItems converts gofeed items to Article models
func (f *Fetcher) convertItems(items []*gofeed.Item, feed *gofeed.Feed, feedURL string) []models.Article {
	articles := make([]models.Article, 0, len(items))
	for _, item := range items {
		article := f.convertToArticle(item, feed, feedURL)
		articles = append(articles, article)
	}
	return articles
}

// convertToArticle converts a gofeed.Item to our Article model.
// Generates a unique ID from the article link using SHA256 hashing.
// Falls back to current time if no publication date is available.
func (f *Fetcher) convertToArticle(item *gofeed.Item, feed *gofeed.Feed, feedURL string) models.Article {
	// Determine publication date with fallback chain: Published > Updated > Now
	published := time.Now()
	if item.PublishedParsed != nil {
		published = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		published = *item.UpdatedParsed
	}

	// Generate deterministic ID from article link using SHA256 hash
	// This ensures the same article always gets the same ID for deduplication
	id := generateID(item.Link)

	author := ""
	if item.Author != nil {
		author = item.Author.Name
	}

	// Use full content if available, otherwise fall back to description
	content := item.Content
	if content == "" {
		content = item.Description
	}

	// Copy categories slice to avoid referencing the original item's slice
	categories := append([]string(nil), item.Categories...)

	return models.Article{
		ID:          id,
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		Content:     content,
		Author:      author,
		Published:   published,
		FeedURL:     feedURL,
		FeedTitle:   feed.Title,
		Categories:  categories,
	}
}

// generateID creates a unique 32-character hex ID from the article link.
// Uses SHA256 hash truncated to 16 bytes (128 bits) for a good balance
// between uniqueness and storage efficiency.
func generateID(link string) string {
	hash := sha256.Sum256([]byte(link))
	return fmt.Sprintf("%x", hash[:16])
}
