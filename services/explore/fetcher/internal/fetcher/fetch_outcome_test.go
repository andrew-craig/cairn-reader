package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/models"
	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch/fetchtest"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/testutil"
)

// recordingFeedRepo is a db.FeedRepositoryInterface stub that hands out one
// feed and captures every UpdateFetchResult / RecordFetchHistory call so a
// test can assert the two always agree.
type recordingFeedRepo struct {
	feed *models.Feed

	updateCalls  []updateFetchResultCall
	historyCalls []recordFetchHistoryCall
}

type updateFetchResultCall struct {
	success      bool
	etag         string
	lastModified string
}

type recordFetchHistoryCall struct {
	success       bool
	articlesFound int
	articlesSent  int
	errMsg        string
}

func (r *recordingFeedRepo) GetNextFeed(context.Context) (*models.Feed, error) { return r.feed, nil }

func (r *recordingFeedRepo) UpdateFetchResult(_ context.Context, _ int, success bool, etag, lastModified string) error {
	r.updateCalls = append(r.updateCalls, updateFetchResultCall{success, etag, lastModified})
	return nil
}

func (r *recordingFeedRepo) RecordFetchHistory(_ context.Context, _ int, success bool, articlesFound, articlesSent int, errMsg string) error {
	r.historyCalls = append(r.historyCalls, recordFetchHistoryCall{success, articlesFound, articlesSent, errMsg})
	return nil
}

func (r *recordingFeedRepo) ImportFeeds(context.Context, []string) error            { return nil }
func (r *recordingFeedRepo) ListFeeds(context.Context, bool) ([]models.Feed, error) { return nil, nil }
func (r *recordingFeedRepo) GetFeedByID(context.Context, int) (*models.Feed, error) { return nil, nil }
func (r *recordingFeedRepo) GetFeedStats(context.Context) (int, int, int, int, error) {
	return 0, 0, 0, 0, nil
}

type stubRecommender struct{ err error }

func (s stubRecommender) SubmitArticles(context.Context, []models.Article) error { return s.err }

// TestFetchSingleFeed_OutcomeRecordedOnce exercises each exit path of
// FetchSingleFeed and asserts UpdateFetchResult and RecordFetchHistory are
// each called exactly once and with a consistent success flag — the property
// the five hand-written recording blocks used to risk violating.
func TestFetchSingleFeed_OutcomeRecordedOnce(t *testing.T) {
	const feedETag = `"srv-etag"`

	tests := []struct {
		name    string
		handler http.HandlerFunc
		feed    models.Feed
		subErr  error
		wantErr bool

		wantUpdate  updateFetchResultCall
		wantHistory recordFetchHistoryCall
	}{
		{
			name:        "fetch error (HTTP 500)",
			handler:     func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantErr:     true,
			wantUpdate:  updateFetchResultCall{success: false},
			wantHistory: recordFetchHistoryCall{success: false},
		},
		{
			name: "304 not modified",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotModified)
			},
			feed:        models.Feed{ETag: feedETag, LastModified: "yesterday"},
			wantUpdate:  updateFetchResultCall{success: true, etag: feedETag, lastModified: "yesterday"},
			wantHistory: recordFetchHistoryCall{success: true},
		},
		{
			name: "parse error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte("not xml at all"))
			},
			wantErr:     true,
			wantUpdate:  updateFetchResultCall{success: false},
			wantHistory: recordFetchHistoryCall{success: false},
		},
		{
			name: "submit error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(testutil.SampleRSSFeed("Feed", 3)))
			},
			subErr:      fmt.Errorf("recommender down"),
			wantErr:     true,
			wantUpdate:  updateFetchResultCall{success: false},
			wantHistory: recordFetchHistoryCall{success: false, articlesFound: 3},
		},
		{
			name: "success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.Header().Set("ETag", feedETag)
				_, _ = w.Write([]byte(testutil.SampleRSSFeed("Feed", 2)))
			},
			wantUpdate:  updateFetchResultCall{success: true, etag: feedETag},
			wantHistory: recordFetchHistoryCall{success: true, articlesFound: 2, articlesSent: 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			feed := tc.feed
			feed.ID = 1
			feed.URL = server.URL
			repo := &recordingFeedRepo{feed: &feed}

			f := NewFetcher(repo, stubRecommender{err: tc.subErr}, time.Minute)
			ctx := fetchtest.AllowLoopback(context.Background())

			err := f.FetchSingleFeed(ctx)
			if tc.wantErr != (err != nil) {
				t.Fatalf("FetchSingleFeed error = %v, wantErr %v", err, tc.wantErr)
			}

			if len(repo.updateCalls) != 1 {
				t.Fatalf("UpdateFetchResult called %d times, want 1", len(repo.updateCalls))
			}
			if len(repo.historyCalls) != 1 {
				t.Fatalf("RecordFetchHistory called %d times, want 1", len(repo.historyCalls))
			}
			if repo.updateCalls[0] != tc.wantUpdate {
				t.Errorf("UpdateFetchResult args = %+v, want %+v", repo.updateCalls[0], tc.wantUpdate)
			}
			gotHistory := repo.historyCalls[0]
			// errMsg carries the live error text on failure paths; compare the
			// rest exactly and assert only presence/absence of an error string.
			gotHistoryErr := gotHistory.errMsg
			gotHistory.errMsg = ""
			if gotHistory != tc.wantHistory {
				t.Errorf("RecordFetchHistory args = %+v, want %+v", gotHistory, tc.wantHistory)
			}
			if tc.wantErr && gotHistoryErr == "" {
				t.Errorf("RecordFetchHistory errMsg empty on a failure path")
			}
			if !tc.wantErr && gotHistoryErr != "" {
				t.Errorf("RecordFetchHistory errMsg = %q on a success path, want empty", gotHistoryErr)
			}
			if repo.updateCalls[0].success != repo.historyCalls[0].success {
				t.Errorf("success flag diverged: update=%v history=%v",
					repo.updateCalls[0].success, repo.historyCalls[0].success)
			}
		})
	}
}
