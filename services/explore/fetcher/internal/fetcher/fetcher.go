// Package fetcher provides RSS feed fetching functionality.
// It fetches feeds from URLs stored in the database, parses RSS/Atom content,
// and submits new articles to the recommender service via HTTP.
package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/models"
	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch"
	"github.com/andrew-craig/cairn-reader/pkg/rss/hash"
	"github.com/andrew-craig/cairn-reader/pkg/rss/parse"
	"github.com/andrew-craig/cairn-reader/pkg/rss/sanitize"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/client"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/db"
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
		// Also the routine post-crash state for up to the lease duration:
		// every enabled feed is currently claimed by another fetch in
		// flight, not necessarily "no feeds configured".
		return fmt.Errorf("no feed currently claimable (none enabled, or all leased)")
	}

	slog.Info("fetching feed", slog.Int("feed_id", feed.ID), slog.String("url", feed.URL))

	resp, err := fetch.Fetch(ctx, feed.URL, fetch.FetchOpts{
		ETag:         feed.ETag,
		LastModified: feed.LastModified,
	})
	if err != nil {
		f.recordOutcome(ctx, feed.ID, fetchOutcome{errMsg: err.Error()})
		return fmt.Errorf("fetch RSS: %w", err)
	}

	// 304 Not Modified — feed unchanged, record success and move on.
	if resp.NotModified {
		slog.Info("feed not modified (304)", slog.Int("feed_id", feed.ID))
		f.recordOutcome(ctx, feed.ID, fetchOutcome{success: true, etag: feed.ETag, lastModified: feed.LastModified})
		return nil
	}

	feedData, err := parse.ParseBytes(resp.Body)
	if err != nil {
		f.recordOutcome(ctx, feed.ID, fetchOutcome{errMsg: err.Error()})
		return fmt.Errorf("parse RSS: %w", err)
	}

	slog.Info("fetched feed", slog.String("title", feedData.Title), slog.Int("items", len(feedData.Items)))

	// Forward every item; the recommender deduplicates by link
	// (article_repository.go: INSERT ... ON CONFLICT (link) DO UPDATE).
	// Filtering by PublishedAt vs. last_fetched_at silently drops items
	// when an upstream CDN serves stale RSS — by the time the cached feed
	// refreshes, last_fetched_at has advanced past the publish timestamp.
	newArticles := f.convertItems(feedData.Items, feedData, feed.URL)

	slog.Info("forwarding articles", slog.Int("count", len(newArticles)), slog.Int("total_items", len(feedData.Items)))

	articlesSent := 0
	if len(newArticles) > 0 {
		if err := f.recommenderClient.SubmitArticles(ctx, newArticles); err != nil {
			slog.Error("failed to submit articles", slog.Any("error", err))
			f.recordOutcome(ctx, feed.ID, fetchOutcome{articlesFound: len(feedData.Items), errMsg: err.Error()})
			return fmt.Errorf("submit articles: %w", err)
		}
		articlesSent = len(newArticles)
		slog.Info("successfully submitted articles", slog.Int("count", articlesSent))
	}

	f.recordOutcome(ctx, feed.ID, fetchOutcome{
		success:       true,
		etag:          resp.ETag,
		lastModified:  resp.LastModified,
		articlesFound: len(feedData.Items),
		articlesSent:  articlesSent,
	})

	return nil
}

// fetchOutcome is the result of one fetch attempt, recorded once via
// recordOutcome. The zero value is a plain failure with no articles seen and
// no conditional-GET values (which UpdateFetchResult ignores on failure).
type fetchOutcome struct {
	success       bool
	etag          string
	lastModified  string
	articlesFound int
	articlesSent  int
	errMsg        string
}

// recordOutcome persists a fetch attempt's result. It is the single place
// UpdateFetchResult and RecordFetchHistory are called, so no exit path can
// record one without the other or let their arguments drift apart.
func (f *Fetcher) recordOutcome(ctx context.Context, feedID int, o fetchOutcome) {
	if err := f.feedRepo.UpdateFetchResult(ctx, feedID, o.success, o.etag, o.lastModified); err != nil {
		slog.Error("failed to update fetch result", slog.Int("feed_id", feedID), slog.Any("error", err))
	}
	if err := f.feedRepo.RecordFetchHistory(ctx, feedID, o.success, o.articlesFound, o.articlesSent, o.errMsg); err != nil {
		slog.Error("failed to record fetch history", slog.Int("feed_id", feedID), slog.Any("error", err))
	}
}

// convertItems converts parse.Items to Article models.
func (f *Fetcher) convertItems(items []parse.Item, feed *parse.Feed, feedURL string) []models.Article {
	articles := make([]models.Article, 0, len(items))
	for _, item := range items {
		articles = append(articles, convertToArticle(item, feed, feedURL))
	}
	return articles
}

// maxContentBytes caps the content payload sent to the recommender.
// Matches fetch.DefaultMaxBodySize so no single item can exceed the ingestion limit.
const maxContentBytes = 5 * 1024 * 1024

// convertToArticle converts a parse.Item to our Article model.
// Article ID is derived from a content hash for consistency with the Read service.
//
// When the feed provides no publish timestamp we emit the zero time rather
// than time.Now(). Items are forwarded on every fetch and the recommender's
// ON CONFLICT (link) DO UPDATE overwrites the published column, so a
// time.Now() fallback would re-stamp dateless items on every cycle and make
// them perpetually float to the top of date-sorted views.
func convertToArticle(item parse.Item, feed *parse.Feed, feedURL string) models.Article {
	var published time.Time
	if item.PublishedAt != nil {
		published = *item.PublishedAt
	}

	// pkg/rss/parse already falls back to Description when Content is absent.
	// Guard against oversized payloads before sanitising.
	content := item.Content
	if content == "" {
		content = item.Description
	}
	if len(content) > maxContentBytes {
		content = content[:maxContentBytes]
	}

	cleanDescription := sanitize.StripHTML(item.Description)
	cleanContent := sanitize.Sanitize(content)

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
