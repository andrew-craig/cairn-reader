// Package fetcher provides RSS feed fetching functionality.
// It fetches feeds from URLs stored in the database, parses RSS/Atom content,
// and submits new articles to the recommender service via HTTP.
package fetcher

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/andrew-craig/cairn-explore/fetcher/internal/client"
	"github.com/andrew-craig/cairn-explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn-explore/pkg/models"
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
}

// NewFetcher creates a new fetcher instance
func NewFetcher(feedRepo *db.FeedRepository, recommenderClient client.RecommenderClientInterface) *Fetcher {
	return &Fetcher{
		feedRepo:          feedRepo,
		recommenderClient: recommenderClient,
		parser:            gofeed.NewParser(),
	}
}

// Run starts the fetch loop that fetches one feed every 60 seconds.
// It performs an initial fetch immediately on startup, then continues
// fetching at the configured interval until the context is cancelled.
// The loop is designed to be gentle on feed sources while ensuring
// all feeds are fetched regularly.
// TODO: Make the interval configurable via constructor parameter
func (f *Fetcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Fetch immediately on startup to begin processing feeds right away
	if err := f.FetchSingleFeed(ctx); err != nil {
		log.Printf("Initial fetch failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := f.FetchSingleFeed(ctx); err != nil {
				log.Printf("Fetch failed: %v", err)
			}
		case <-ctx.Done():
			log.Println("Stopping fetcher")
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

	log.Printf("Fetching feed %d: %s", feed.ID, feed.URL)

	// Step 2: Fetch and parse RSS/Atom content from the feed URL
	feedData, err := f.fetchRSS(ctx, feed.URL)
	if err != nil {
		if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false); updateErr != nil {
			log.Printf("error updating fetch result for feed %d: %v", feed.ID, updateErr)
		}
		if histErr := f.feedRepo.RecordFetchHistory(ctx, feed.ID, false, 0, 0, err.Error()); histErr != nil {
			log.Printf("error recording fetch history for feed %d: %v", feed.ID, histErr)
		}
		return fmt.Errorf("fetch RSS: %w", err)
	}

	log.Printf("Fetched feed: %s (%d items)", feedData.Title, len(feedData.Items))

	// Step 3: Filter to only include articles published after last successful fetch
	newArticles := f.filterNewArticles(feedData.Items, feed.LastFetchedAt, feedData, feed.URL)

	log.Printf("Found %d new articles (total items: %d)", len(newArticles), len(feedData.Items))

	// Step 4: Submit new articles to the recommender service via HTTP POST
	articlesSent := 0
	if len(newArticles) > 0 {
		if err := f.recommenderClient.SubmitArticles(ctx, newArticles); err != nil {
			log.Printf("Failed to submit articles: %v", err)
			if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false); updateErr != nil {
				log.Printf("error updating fetch result for feed %d: %v", feed.ID, updateErr)
			}
			if histErr := f.feedRepo.RecordFetchHistory(ctx, feed.ID, false, len(feedData.Items), 0, err.Error()); histErr != nil {
				log.Printf("error recording fetch history for feed %d: %v", feed.ID, histErr)
			}
			return fmt.Errorf("submit articles: %w", err)
		}
		articlesSent = len(newArticles)
		log.Printf("Successfully submitted %d articles", articlesSent)
	}

	// Step 5: Record successful fetch - resets consecutive failure count
	success := true // Success if we completed the fetch, even if no new articles
	if err := f.feedRepo.UpdateFetchResult(ctx, feed.ID, success); err != nil {
		log.Printf("error updating fetch result for feed %d: %v", feed.ID, err)
	}
	if err := f.feedRepo.RecordFetchHistory(ctx, feed.ID, success, len(feedData.Items), articlesSent, ""); err != nil {
		log.Printf("error recording fetch history for feed %d: %v", feed.ID, err)
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
