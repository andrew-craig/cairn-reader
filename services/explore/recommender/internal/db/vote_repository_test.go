package db

import (
	"context"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/models"
)

// TestGetUserVoteStats_Aggregate verifies that GetUserVoteStats returns correct counts
// using an aggregate SQL query (no client-side counting).
// This test is skipped when the test database is unavailable.
func TestGetUserVoteStats_Aggregate(t *testing.T) {
	dbPool := setupTestDB(t)
	defer teardownTestDB(t, dbPool)

	ctx := context.Background()

	// Create the article and user repositories (vote repo depends on user repo)
	userRepo := NewUserRepository(dbPool)
	voteRepo := NewVoteRepository(dbPool, userRepo)
	articleRepo := NewArticleRepository(dbPool)

	// Seed two articles
	art1 := createTestArticle("vote-stat-art-1", "https://example.com/vs1", "VS Article 1")
	art1.Published = time.Now()
	art2 := createTestArticle("vote-stat-art-2", "https://example.com/vs2", "VS Article 2")
	art2.Published = time.Now()
	for _, a := range []models.Article{art1, art2} {
		if err := articleRepo.Create(ctx, a); err != nil {
			t.Fatalf("failed to seed article: %v", err)
		}
	}

	userID := "test-vote-stats-user"

	// Cast: 1 upvote, 1 downvote
	if err := voteRepo.RecordVote(ctx, userID, art1.ID, "upvote"); err != nil {
		t.Fatalf("RecordVote upvote art1: %v", err)
	}
	if err := voteRepo.RecordVote(ctx, userID, art2.ID, "downvote"); err != nil {
		t.Fatalf("RecordVote downvote art2: %v", err)
	}

	upvotes, downvotes, err := voteRepo.GetUserVoteStats(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserVoteStats returned error: %v", err)
	}

	if upvotes != 1 {
		t.Errorf("expected 1 upvote, got %d", upvotes)
	}
	if downvotes != 1 {
		t.Errorf("expected 1 downvote, got %d", downvotes)
	}
}

// TestGetUserVoteStats_NoVotes verifies that a user with no votes returns zeros.
func TestGetUserVoteStats_NoVotes(t *testing.T) {
	dbPool := setupTestDB(t)
	defer teardownTestDB(t, dbPool)

	ctx := context.Background()
	userRepo := NewUserRepository(dbPool)
	voteRepo := NewVoteRepository(dbPool, userRepo)

	// User who has never voted
	upvotes, downvotes, err := voteRepo.GetUserVoteStats(ctx, "never-voted-user")
	if err != nil {
		t.Fatalf("GetUserVoteStats returned error for unknown user: %v", err)
	}
	if upvotes != 0 || downvotes != 0 {
		t.Errorf("expected (0,0) for unknown user, got (%d,%d)", upvotes, downvotes)
	}
}
