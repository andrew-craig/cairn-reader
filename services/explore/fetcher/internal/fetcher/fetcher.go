// Package fetcher provides RSS feed fetching functionality.
// It fetches feeds from URLs stored in the database, parses RSS/Atom content,
// and submits new articles to the recommender service via HTTP.
package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/pkg/rss/fetch"
	"github.com/cairn-app/cairn-reader/pkg/rss/hash"
	"github.com/cairn-app/cairn-reader/pkg/rss/parse"
	"github.com/cairn-app/cairn-reader/pkg/rss/sanitize"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/client"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
)

// Fetcher handles RSS feed fetching and article submission.
type Fetcher struct {
	feedRepo          db.FeedRepositoryInterface
	recommenderClient client.RecommenderClientInterface
	fetchInterval     time.Duration
}

// NewFetcher creates a new fetcher instance with the specified fetch interval.
func NewFetcher(feedRepo db.FeedRepositoryInterface, recommenderClient client.RecommenderClientInterface, fetchInterval time.Duration) *Fetcher {
	return &Fetcher{
		feedRepo:          feedRepo,
		recommenderClient: recommenderClient,
		fetchInterval:     fetchInterval,
	}
}

// Run starts the fetch loop that fetches one feed at the configured interval.
func (f *Fetcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	slog.Info("starting fetcher", slog.Duration("interval", f.fetchInterval))

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

// FetchSingleFeed fetches one feed and sends new articles to the recommender.
func (f *Fetcher) FetchSingleFeed(ctx context.Context) error {
	feed, err := f.feedRepo.GetNextFeed(ctx)
	if err != nil {
		return fmt.Errorf("get next feed: %w", err)
	}
	if feed == nil {
		return fmt.Errorf("no enabled feeds available")
	}

	slog.Info("fetching feed", slog.Int("feed_id", feed.ID), slog.String("url", feed.URL))

	resp, err := fetch.Fetch(ctx, feed.URL, fetch.FetchOpts{
		ETag:         feed.ETag,
		LastModified: feed.LastModified,
	})
	if err != nil {
		if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false, "", ""); updateErr != nil {
			slog.Error("failed to update fetch result", slog.Int("feed_id", feed.ID), slog.Any("error", updateErr))
		}
		if histErr := f.feedRepo.RecordFetchHistory(ctx, feed.ID, false, 0, 0, err.Error()); histErr != nil {
			slog.Error("failed to record fetch history", slog.Int("feed_id", feed.ID), slog.Any("error", histErr))
		}
		return fmt.Errorf("fetch RSS: %w", err)
	}

	// 304 Not Modified — feed unchanged, record success and move on.
	if resp.NotModified {
		slog.Info("feed not modified (304)", slog.Int("feed_id", feed.ID))
		if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, true, feed.ETag, feed.LastModified); updateErr != nil {
			slog.Error("failed to update fetch result", slog.Int("feed_id", feed.ID), slog.Any("error", updateErr))
		}
		if histErr := f.feedRepo.RecordFetchHistory(ctx, feed.ID, true, 0, 0, ""); histErr != nil {
			slog.Error("failed to record fetch history", slog.Int("feed_id", feed.ID), slog.Any("error", histErr))
		}
		return nil
	}

	feedData, err := parse.ParseBytes(resp.Body)
	if err != nil {
		if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false, "", ""); updateErr != nil {
			slog.Error("failed to update fetch result", slog.Int("feed_id", feed.ID), slog.Any("error", updateErr))
		}
		if histErr := f.feedRepo.RecordFetchHistory(ctx, feed.ID, false, 0, 0, err.Error()); histErr != nil {
			slog.Error("failed to record fetch history", slog.Int("feed_id", feed.ID), slog.Any("error", histErr))
		}
		return fmt.Errorf("parse RSS: %w", err)
	}

	slog.Info("fetched feed", slog.String("title", feedData.Title), slog.Int("items", len(feedData.Items)))

	newArticles := f.filterNewArticles(feedData.Items, feed.LastFetchedAt, feedData, feed.URL)

	slog.Info("found new articles", slog.Int("new_count", len(newArticles)), slog.Int("total_items", len(feedData.Items)))

	articlesSent := 0
	if len(newArticles) > 0 {
		if err := f.recommenderClient.SubmitArticles(ctx, newArticles); err != nil {
			slog.Error("failed to submit articles", slog.Any("error", err))
			if updateErr := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false, "", ""); updateErr != nil {
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

	if err := f.feedRepo.UpdateFetchResult(ctx, feed.ID, true, resp.ETag, resp.LastModified); err != nil {
		slog.Error("failed to update fetch result", slog.Int("feed_id", feed.ID), slog.Any("error", err))
	}
	if err := f.feedRepo.RecordFetchHistory(ctx, feed.ID, true, len(feedData.Items), articlesSent, ""); err != nil {
		slog.Error("failed to record fetch history", slog.Int("feed_id", feed.ID), slog.Any("error", err))
	}

	return nil
}

// filterNewArticles returns only articles published after lastFetch.
// For the first fetch of a feed (lastFetch is nil), all articles are returned.
func (f *Fetcher) filterNewArticles(items []parse.Item, lastFetch *time.Time, feed *parse.Feed, feedURL string) []models.Article {
	if lastFetch == nil {
		return f.convertItems(items, feed, feedURL)
	}

	var newItems []parse.Item
	for _, item := range items {
		if item.PublishedAt != nil && item.PublishedAt.After(*lastFetch) {
			newItems = append(newItems, item)
		}
	}
	return f.convertItems(newItems, feed, feedURL)
}

// convertItems converts parse.Items to Article models.
func (f *Fetcher) convertItems(items []parse.Item, feed *parse.Feed, feedURL string) []models.Article {
	articles := make([]models.Article, 0, len(items))
	for _, item := range items {
		articles = append(articles, convertToArticle(item, feed, feedURL))
	}
	return articles
}

// convertToArticle converts a parse.Item to our Article model.
// Article ID is derived from a content hash for consistency with the Read service.
func convertToArticle(item parse.Item, feed *parse.Feed, feedURL string) models.Article {
	published := time.Now()
	if item.PublishedAt != nil {
		published = *item.PublishedAt
	}

	cleanDescription := sanitize.StripHTML(item.Description)
	cleanContent := sanitize.Sanitize(item.Content)

	id := hash.ContentHash([]byte(cleanContent))

	return models.Article{
		ID:          id,
		Title:       item.Title,
		Link:        item.Link,
		Description: cleanDescription,
		Content:     cleanContent,
		Author:      item.Author,
		Published:   published,
		FeedURL:     feedURL,
		FeedTitle:   feed.Title,
	}
}
