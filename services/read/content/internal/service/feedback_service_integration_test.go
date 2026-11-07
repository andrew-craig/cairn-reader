//go:build integration
// +build integration

package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type feedbackSnapshotRow struct {
	status            string
	rawHTML           sql.NullString
	cleanedHTML       sql.NullString
	readabilityFailed sql.NullBool
	imageURL          sql.NullString
	title             string
}

func readFeedbackSnapshot(t *testing.T, db *sql.DB, userID, contentID uuid.UUID) feedbackSnapshotRow {
	t.Helper()
	var row feedbackSnapshotRow
	require.NoError(t, db.QueryRow(`
		SELECT status, raw_html, cleaned_html, readability_failed, image_url, title
		FROM content_feedback WHERE user_id = $1 AND content_id = $2`, userID, contentID,
	).Scan(&row.status, &row.rawHTML, &row.cleanedHTML, &row.readabilityFailed, &row.imageURL, &row.title))
	return row
}

func TestFeedbackService_Submit_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	ctx := context.Background()

	contentRepo := repository.NewContentRepository(testDB.DB)
	userContentRepo := repository.NewUserContentRepository(testDB.DB)
	feedbackRepo := repository.NewContentFeedbackRepository(testDB.DB)
	svc := NewFeedbackService(userContentRepo, contentRepo, feedbackRepo)

	userID := uuid.New()
	content := &models.Content{
		ContentHash: "h1",
		CleanedHTML: "<p>cleaned</p>",
		OriginalURL: "https://example.com/a",
		Title:       "Title",
		SourceType:  models.SourceTypeWeb,
		ImageURLs:   pq.StringArray{"https://example.com/lead.png", "https://example.com/second.png"},
		Source:      &models.ContentSource{RawHTML: "<html>raw</html>", ReadabilityFailed: true},
	}
	require.NoError(t, contentRepo.Create(ctx, content))
	require.NoError(t, userContentRepo.Create(ctx, &models.UserContent{
		UserID: userID, ContentID: content.ID, Status: models.StatusUnread, List: models.ListReads,
	}))

	t.Run("not owned", func(t *testing.T) {
		_, err := svc.Submit(ctx, uuid.New(), content.ID, []string{"images"}, nil)
		require.ErrorIs(t, err, ErrUserContentNotFound)
	})

	t.Run("creates the report with a snapshot including the raw source", func(t *testing.T) {
		comment := "no images"
		saved, err := svc.Submit(ctx, userID, content.ID, []string{"images", "lead_image"}, &comment)
		require.NoError(t, err)
		require.Equal(t, models.FeedbackStatusOpen, saved.Status)
		require.Equal(t, pq.StringArray{"images", "lead_image"}, saved.Categories)
		require.Equal(t, &comment, saved.Comment)
		require.Nil(t, saved.Snapshot, "stored snapshot is never returned")

		got, err := feedbackRepo.GetByUserAndContent(ctx, userID, content.ID)
		require.NoError(t, err)
		require.Equal(t, saved.ID, got.ID)
		require.Nil(t, got.Snapshot)

		row := readFeedbackSnapshot(t, testDB.DB, userID, content.ID)
		require.Equal(t, "<html>raw</html>", row.rawHTML.String)
		require.Equal(t, "<p>cleaned</p>", row.cleanedHTML.String)
		require.True(t, row.readabilityFailed.Valid)
		require.True(t, row.readabilityFailed.Bool)
		require.Equal(t, "https://example.com/lead.png", row.imageURL.String)
		require.Equal(t, "Title", row.title)
	})

	t.Run("no report", func(t *testing.T) {
		got, err := feedbackRepo.GetByUserAndContent(ctx, userID, uuid.New())
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("expired source leaves raw_html and readability_failed NULL", func(t *testing.T) {
		_, err := testDB.DB.Exec(`DELETE FROM content_sources WHERE content_id = $1`, content.ID)
		require.NoError(t, err)

		_, err = svc.Submit(ctx, userID, content.ID, []string{"title"}, nil)
		require.NoError(t, err)

		row := readFeedbackSnapshot(t, testDB.DB, userID, content.ID)
		require.False(t, row.rawHTML.Valid)
		require.False(t, row.readabilityFailed.Valid)
		require.Equal(t, "<p>cleaned</p>", row.cleanedHTML.String)
	})

	t.Run("absent image leaves image_url NULL", func(t *testing.T) {
		noImage := &models.Content{
			ContentHash: "h2", CleanedHTML: "<p>x</p>", OriginalURL: "https://example.com/b",
			Title: "No image", SourceType: models.SourceTypeWeb,
		}
		require.NoError(t, contentRepo.Create(ctx, noImage))
		require.NoError(t, userContentRepo.Create(ctx, &models.UserContent{
			UserID: userID, ContentID: noImage.ID, Status: models.StatusUnread, List: models.ListReads,
		}))

		_, err := svc.Submit(ctx, userID, noImage.ID, []string{"other"}, nil)
		require.NoError(t, err)
		require.False(t, readFeedbackSnapshot(t, testDB.DB, userID, noImage.ID).imageURL.Valid)
	})
}

func TestFeedbackService_Resubmit_ReopensAndRefreshesSnapshot_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	ctx := context.Background()

	contentRepo := repository.NewContentRepository(testDB.DB)
	userContentRepo := repository.NewUserContentRepository(testDB.DB)
	feedbackRepo := repository.NewContentFeedbackRepository(testDB.DB)
	svc := NewFeedbackService(userContentRepo, contentRepo, feedbackRepo)

	userID := uuid.New()
	content := &models.Content{
		ContentHash: "h1", CleanedHTML: "<p>old</p>", OriginalURL: "https://example.com/a",
		Title: "Old title", SourceType: models.SourceTypeWeb,
		Source: &models.ContentSource{RawHTML: "<html>old</html>", ReadabilityFailed: false},
	}
	require.NoError(t, contentRepo.Create(ctx, content))
	require.NoError(t, userContentRepo.Create(ctx, &models.UserContent{
		UserID: userID, ContentID: content.ID, Status: models.StatusUnread, List: models.ListReads,
	}))

	first, err := svc.Submit(ctx, userID, content.ID, []string{"formatting"}, nil)
	require.NoError(t, err)

	// Triage resolved it: status flips and the bodies are cleared.
	_, err = testDB.DB.Exec(`UPDATE content_feedback SET status = 'resolved', raw_html = NULL, cleaned_html = NULL WHERE id = $1`, first.ID)
	require.NoError(t, err)

	// The content is re-extracted, then the user reports again.
	content.CleanedHTML = "<p>new</p>"
	content.Title = "New title"
	content.Source = &models.ContentSource{RawHTML: "<html>new</html>", ReadabilityFailed: true}
	require.NoError(t, contentRepo.Update(ctx, content))

	comment := "still wrong"
	second, err := svc.Submit(ctx, userID, content.ID, []string{"body_missing_content", "links"}, &comment)
	require.NoError(t, err)

	require.Equal(t, first.ID, second.ID, "resubmitting updates the same report")
	require.Equal(t, models.FeedbackStatusOpen, second.Status)
	require.Equal(t, pq.StringArray{"body_missing_content", "links"}, second.Categories)
	require.Equal(t, &comment, second.Comment)
	require.True(t, second.CreatedAt.Equal(first.CreatedAt), "created_at is kept")
	require.True(t, second.UpdatedAt.After(first.UpdatedAt))

	row := readFeedbackSnapshot(t, testDB.DB, userID, content.ID)
	require.Equal(t, "open", row.status)
	require.Equal(t, "<html>new</html>", row.rawHTML.String)
	require.Equal(t, "<p>new</p>", row.cleanedHTML.String)
	require.True(t, row.readabilityFailed.Bool)
	require.Equal(t, "New title", row.title)

	var count int
	require.NoError(t, testDB.DB.QueryRow(`SELECT count(*) FROM content_feedback`).Scan(&count))
	require.Equal(t, 1, count)
}
