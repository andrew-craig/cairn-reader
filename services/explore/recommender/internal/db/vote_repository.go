// Package db provides database access for the recommender service.
// It implements the repository pattern for articles, users, and votes.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// VoteRepository handles vote database operations
type VoteRepository struct {
	db             *sql.DB
	userRepository *UserRepository
}

// NewVoteRepository creates a new vote repository
func NewVoteRepository(db *sql.DB, userRepo *UserRepository) *VoteRepository {
	return &VoteRepository{
		db:             db,
		userRepository: userRepo,
	}
}

// RecordVote inserts or updates a vote (upsert)
// Updates articles.upvotes and articles.downvotes counts atomically
// Implements Phase 3 voting logic with user auto-creation
func (r *VoteRepository) RecordVote(ctx context.Context, userID string, articleID string, voteType string) error {
	// Validate vote type
	if voteType != "upvote" && voteType != "downvote" {
		return fmt.Errorf("invalid vote type: %s (must be 'upvote' or 'downvote')", voteType)
	}

	// Ensure user exists
	if err := r.userRepository.EnsureUserExists(ctx, userID); err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Start a transaction to ensure atomicity
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			slog.Error("failed to rollback transaction", slog.Any("error", err))
		}
	}()

	// Check if user has already voted on this article
	var existingVoteType sql.NullString
	checkQuery := `SELECT vote_type FROM votes WHERE user_id = $1 AND article_id = $2`
	err = tx.QueryRowContext(ctx, checkQuery, userID, articleID).Scan(&existingVoteType)
	if err != nil && err != sql.ErrNoRows {
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

		result, err := tx.ExecContext(ctx, updateQuery, articleID)
		if err != nil {
			return fmt.Errorf("failed to update article vote counts: %w", err)
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			slog.Warn("vote counter update had no effect (possibly already at 0)",
				slog.String("article_id", articleID),
				slog.String("old_vote_type", oldVoteType),
				slog.String("new_vote_type", voteType),
			)
			return fmt.Errorf("article not found: %s", articleID)
		}

		// Update the vote record
		voteQuery := `
			UPDATE votes
			SET vote_type = $1
			WHERE user_id = $2 AND article_id = $3
		`
		_, err = tx.ExecContext(ctx, voteQuery, voteType, userID, articleID)
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

		result, err := tx.ExecContext(ctx, updateQuery, articleID)
		if err != nil {
			return fmt.Errorf("failed to update article vote counts: %w", err)
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("article not found: %s", articleID)
		}

		// Insert the vote record
		voteQuery := `
			INSERT INTO votes (user_id, article_id, vote_type, created_at)
			VALUES ($1, $2, $3, NOW())
		`
		_, err = tx.ExecContext(ctx, voteQuery, userID, articleID, voteType)
		if err != nil {
			return fmt.Errorf("failed to insert vote: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RemoveVote deletes a vote and updates article counts
func (r *VoteRepository) RemoveVote(ctx context.Context, userID string, articleID string) error {

	// Start a transaction to ensure atomicity
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			slog.Error("failed to rollback transaction", slog.Any("error", err))
		}
	}()

	// Get the existing vote type
	var voteType string
	checkQuery := `SELECT vote_type FROM votes WHERE user_id = $1 AND article_id = $2`
	err = tx.QueryRowContext(ctx, checkQuery, userID, articleID).Scan(&voteType)
	if err == sql.ErrNoRows {
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

	result, err := tx.ExecContext(ctx, updateQuery, articleID)
	if err != nil {
		return fmt.Errorf("failed to update article vote counts: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		slog.Warn("vote counter update had no effect (possibly already at 0)",
			slog.String("article_id", articleID),
			slog.String("vote_type", voteType),
		)
	}

	// Delete the vote record
	deleteQuery := `DELETE FROM votes WHERE user_id = $1 AND article_id = $2`
	_, err = tx.ExecContext(ctx, deleteQuery, userID, articleID)
	if err != nil {
		return fmt.Errorf("failed to delete vote: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetVoteCounts returns upvote/downvote counts for an article
func (r *VoteRepository) GetVoteCounts(ctx context.Context, articleID string) (upvotes int, downvotes int, err error) {
	query := `SELECT upvotes, downvotes FROM articles WHERE id = $1`

	err = r.db.QueryRowContext(ctx, query, articleID).Scan(&upvotes, &downvotes)
	if err == sql.ErrNoRows {
		return 0, 0, fmt.Errorf("article not found")
	}
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get vote counts: %w", err)
	}

	return upvotes, downvotes, nil
}

// GetUserVote returns the user's vote for an article (if any)
// Returns empty string if user hasn't voted
func (r *VoteRepository) GetUserVote(ctx context.Context, userID string, articleID string) (voteType string, err error) {
	query := `SELECT vote_type FROM votes WHERE user_id = $1 AND article_id = $2`

	err = r.db.QueryRowContext(ctx, query, userID, articleID).Scan(&voteType)
	if err == sql.ErrNoRows {
		return "", nil // User hasn't voted
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user vote: %w", err)
	}

	return voteType, nil
}
