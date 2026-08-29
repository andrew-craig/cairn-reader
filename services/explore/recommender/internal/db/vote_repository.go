// Package db provides database access for the recommender service.
// It implements the repository pattern for articles, users, and votes.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	apperrors "github.com/andrew-craig/cairn-reader/pkg/errors"
	"github.com/andrew-craig/cairn-reader/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// voteRepository handles vote database operations
type voteRepository struct {
	db             *pgxpool.Pool
	userRepository UserRepositoryInterface
}

// NewVoteRepository creates a new vote repository
func NewVoteRepository(db *pgxpool.Pool, userRepo UserRepositoryInterface) VoteRepositoryInterface {
	return &voteRepository{
		db:             db,
		userRepository: userRepo,
	}
}

// RecordVote inserts or updates a vote (upsert)
// Updates articles.upvotes and articles.downvotes counts atomically
// Implements Phase 3 voting logic with user auto-creation
func (r *voteRepository) RecordVote(ctx context.Context, userID string, articleID string, voteType string) error {
	// Validate vote type
	if voteType != "upvote" && voteType != "downvote" {
		return fmt.Errorf("invalid vote type %s (must be 'upvote' or 'downvote'): %w", voteType, apperrors.ErrInvalidVoteType)
	}

	// Ensure user exists
	if err := r.userRepository.EnsureUserExists(ctx, userID); err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Start a transaction to ensure atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Error("failed to rollback transaction", slog.Any("error", err))
		}
	}()

	// Check if user has already voted on this article
	var existingVoteType sql.NullString
	checkQuery := `SELECT vote_type FROM votes WHERE user_id = $1 AND article_id = $2`
	err = tx.QueryRow(ctx, checkQuery, userID, articleID).Scan(&existingVoteType)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to check existing vote: %w", err)
	}

	// Update article vote counts based on the vote change
	if existingVoteType.Valid {
		// User is changing their vote
		if existingVoteType.String == voteType {
			// Same vote type, nothing to do
			return nil
		}

		// Different vote type: decrement old, increment new
		var updateQuery string
		var oldVoteType string
		if existingVoteType.String == "upvote" && voteType == "downvote" {
			updateQuery = `
				UPDATE articles
				SET upvotes = upvotes - 1, downvotes = downvotes + 1, updated_at = NOW()
				WHERE id = $1 AND upvotes > 0
			`
			oldVoteType = "upvote"
		} else { // existing is downvote, new is upvote
			updateQuery = `
				UPDATE articles
				SET upvotes = upvotes + 1, downvotes = downvotes - 1, updated_at = NOW()
				WHERE id = $1 AND downvotes > 0
			`
			oldVoteType = "downvote"
		}

		result, err := tx.Exec(ctx, updateQuery, articleID)
		if err != nil {
			return fmt.Errorf("failed to update article vote counts: %w", err)
		}
		if result.RowsAffected() == 0 {
			slog.Warn("vote counter update had no effect (possibly already at 0)",
				slog.String("article_id", articleID),
				slog.String("old_vote_type", oldVoteType),
				slog.String("new_vote_type", voteType),
			)
			return fmt.Errorf("article %s: %w", articleID, apperrors.ErrArticleNotFound)
		}

		// Update the vote record
		voteQuery := `
			UPDATE votes
			SET vote_type = $1
			WHERE user_id = $2 AND article_id = $3
		`
		_, err = tx.Exec(ctx, voteQuery, voteType, userID, articleID)
		if err != nil {
			return fmt.Errorf("failed to update vote: %w", err)
		}
	} else {
		// New vote: increment the appropriate counter
		var updateQuery string
		if voteType == "upvote" {
			updateQuery = `
				UPDATE articles
				SET upvotes = upvotes + 1, updated_at = NOW()
				WHERE id = $1
			`
		} else {
			updateQuery = `
				UPDATE articles
				SET downvotes = downvotes + 1, updated_at = NOW()
				WHERE id = $1
			`
		}

		result, err := tx.Exec(ctx, updateQuery, articleID)
		if err != nil {
			return fmt.Errorf("failed to update article vote counts: %w", err)
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("article %s: %w", articleID, apperrors.ErrArticleNotFound)
		}

		// Insert the vote record
		voteQuery := `
			INSERT INTO votes (user_id, article_id, vote_type, created_at)
			VALUES ($1, $2, $3, NOW())
		`
		_, err = tx.Exec(ctx, voteQuery, userID, articleID, voteType)
		if err != nil {
			return fmt.Errorf("failed to insert vote: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RemoveVote deletes a vote and updates article counts
func (r *voteRepository) RemoveVote(ctx context.Context, userID string, articleID string) error {

	// Start a transaction to ensure atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Error("failed to rollback transaction", slog.Any("error", err))
		}
	}()

	// Get the existing vote type
	var voteType string
	checkQuery := `SELECT vote_type FROM votes WHERE user_id = $1 AND article_id = $2`
	err = tx.QueryRow(ctx, checkQuery, userID, articleID).Scan(&voteType)
	if err == pgx.ErrNoRows {
		// No vote to remove
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check existing vote: %w", err)
	}

	// Decrement the appropriate counter
	var updateQuery string
	if voteType == "upvote" {
		updateQuery = `
			UPDATE articles
			SET upvotes = upvotes - 1, updated_at = NOW()
			WHERE id = $1 AND upvotes > 0
		`
	} else {
		updateQuery = `
			UPDATE articles
			SET downvotes = downvotes - 1, updated_at = NOW()
			WHERE id = $1 AND downvotes > 0
		`
	}

	result, err := tx.Exec(ctx, updateQuery, articleID)
	if err != nil {
		return fmt.Errorf("failed to update article vote counts: %w", err)
	}
	if result.RowsAffected() == 0 {
		slog.Warn("vote counter update had no effect (possibly already at 0)",
			slog.String("article_id", articleID),
			slog.String("vote_type", voteType),
		)
	}

	// Delete the vote record
	deleteQuery := `DELETE FROM votes WHERE user_id = $1 AND article_id = $2`
	_, err = tx.Exec(ctx, deleteQuery, userID, articleID)
	if err != nil {
		return fmt.Errorf("failed to delete vote: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetVoteCounts returns upvote/downvote counts for an article
func (r *voteRepository) GetVoteCounts(ctx context.Context, articleID string) (upvotes int, downvotes int, err error) {
	query := `SELECT upvotes, downvotes FROM articles WHERE id = $1`

	err = r.db.QueryRow(ctx, query, articleID).Scan(&upvotes, &downvotes)
	if err == pgx.ErrNoRows {
		return 0, 0, apperrors.ErrArticleNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get vote counts: %w", err)
	}

	return upvotes, downvotes, nil
}

// GetUserVote returns the user's vote for an article (if any)
// Returns empty string if user hasn't voted
func (r *voteRepository) GetUserVote(ctx context.Context, userID string, articleID string) (voteType string, err error) {
	query := `SELECT vote_type FROM votes WHERE user_id = $1 AND article_id = $2`

	err = r.db.QueryRow(ctx, query, userID, articleID).Scan(&voteType)
	if err == pgx.ErrNoRows {
		return "", nil // User hasn't voted
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user vote: %w", err)
	}

	return voteType, nil
}

// GetUserVoteStats returns aggregate upvote/downvote counts for the user using a single
// COUNT ... FILTER query — no row fetching, no client-side counting.
func (r *voteRepository) GetUserVoteStats(ctx context.Context, userID string) (upvotes int, downvotes int, err error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE vote_type = 'upvote')   AS upvotes,
			COUNT(*) FILTER (WHERE vote_type = 'downvote') AS downvotes
		FROM votes
		WHERE user_id = $1
	`
	err = r.db.QueryRow(ctx, query, userID).Scan(&upvotes, &downvotes)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get user vote stats: %w", err)
	}
	return upvotes, downvotes, nil
}

// GetUserVotedArticles returns all articles a user has voted on with their vote types
// Results are ordered by vote creation time (most recent first)
func (r *voteRepository) GetUserVotedArticles(ctx context.Context, userID string, limit int, offset int) ([]VotedArticle, error) {
	query := `
		SELECT
			a.id, a.title, a.link, a.description, a.content, a.author,
			a.published, a.feed_url, a.feed_title, a.categories,
			a.upvotes, a.downvotes, a.recommends, a.deleted,
			a.created_at, a.updated_at,
			v.vote_type
		FROM votes v
		JOIN articles a ON v.article_id = a.id
		WHERE v.user_id = $1 AND a.deleted = false
		ORDER BY v.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query user voted articles: %w", err)
	}
	defer rows.Close()

	// Initialize as empty slice to return [] instead of null when no votes exist
	votedArticles := make([]VotedArticle, 0)
	for rows.Next() {
		var article models.Article
		var voteType string

		err := rows.Scan(
			&article.ID, &article.Title, &article.Link, &article.Description, &article.Content, &article.Author,
			&article.Published, &article.FeedURL, &article.FeedTitle, &article.Categories,
			&article.Upvotes, &article.Downvotes, &article.Recommends, &article.Deleted,
			&article.CreatedAt, &article.UpdatedAt,
			&voteType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan voted article: %w", err)
		}

		votedArticles = append(votedArticles, VotedArticle{
			Article:  article,
			VoteType: voteType,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating voted articles: %w", err)
	}

	return votedArticles, nil
}
