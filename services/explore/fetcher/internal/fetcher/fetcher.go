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

// Fetcher handles RSS feed fetching
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

// Run starts the fetch loop (1 feed per minute)
func (f *Fetcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Fetch immediately on startup
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

// FetchSingleFeed fetches one feed and sends articles to recommender
func (f *Fetcher) FetchSingleFeed(ctx context.Context) error {
	// 1. Get next feed from database
	feed, err := f.feedRepo.GetNextFeed(ctx)
	if err != nil {
		return fmt.Errorf("get next feed: %w", err)
	}
	if feed == nil {
		return fmt.Errorf("no enabled feeds available")
	}

	log.Printf("Fetching feed %d: %s", feed.ID, feed.URL)

	// 2. Fetch RSS content
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

	// 3. Filter new articles (only published after last fetch)
	newArticles := f.filterNewArticles(feedData.Items, feed.LastFetchedAt, feedData, feed.URL)

	log.Printf("Found %d new articles (total items: %d)", len(newArticles), len(feedData.Items))

	// 4. Send articles to recommender
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

	// 5. Update fetch status
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

// filterNewArticles returns only articles published after lastFetch
func (f *Fetcher) filterNewArticles(items []*gofeed.Item, lastFetch *time.Time, feed *gofeed.Feed, feedURL string) []models.Article {
	if lastFetch == nil {
		// First fetch, send all articles
		return f.convertItems(items, feed, feedURL)
	}

	var newItems []*gofeed.Item
	for _, item := range items {
		// Check PublishedParsed first, then UpdatedParsed
		if item.PublishedParsed != nil && item.PublishedParsed.After(*lastFetch) {
			newItems = append(newItems, item)
		} else if item.PublishedParsed == nil && item.UpdatedParsed != nil && item.UpdatedParsed.After(*lastFetch) {
			// Use UpdatedParsed only if PublishedParsed is not available
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

// convertToArticle converts a gofeed.Item to our Article model
func (c *Fetcher) convertToArticle(item *gofeed.Item, feed *gofeed.Feed, feedURL string) models.Article {
	published := time.Now()
	if item.PublishedParsed != nil {
		published = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		published = *item.UpdatedParsed
	}

	// Generate a unique ID based on the link
	id := generateID(item.Link)

	author := ""
	if item.Author != nil {
		author = item.Author.Name
	}

	content := item.Content
	if content == "" {
		content = item.Description
	}

	categories := make([]string, 0, len(item.Categories))
	for _, cat := range item.Categories {
		categories = append(categories, cat)
	}

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

// generateID creates a unique ID from the article link
func generateID(link string) string {
	hash := sha256.Sum256([]byte(link))
	return fmt.Sprintf("%x", hash[:16])
}
