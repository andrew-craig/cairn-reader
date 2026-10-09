package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/google/uuid"
)

// SourceRouteRepository stores each user's chosen destination list per source.
// A source with no route delivers to Reads.
type SourceRouteRepository interface {
	// Upsert creates or replaces the route for (user, source type, source key).
	Upsert(ctx context.Context, route *models.SourceRoute) error

	// Delete removes a route. Deleting a route that does not exist is not an error.
	Delete(ctx context.Context, userID uuid.UUID, sourceType string, sourceKey uuid.UUID) error

	// ListByUser returns all of a user's routes.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.SourceRoute, error)
}

type sourceRouteRepository struct {
	db *sql.DB
}

// NewSourceRouteRepository creates a new SourceRouteRepository
func NewSourceRouteRepository(db *sql.DB) SourceRouteRepository {
	return &sourceRouteRepository{db: db}
}

func (r *sourceRouteRepository) Upsert(ctx context.Context, route *models.SourceRoute) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO source_routes (user_id, source_type, source_key, list)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, source_type, source_key)
		DO UPDATE SET list = EXCLUDED.list, updated_at = NOW()
	`, route.UserID, route.SourceType, route.SourceKey, route.List)
	if err != nil {
		return fmt.Errorf("failed to upsert source route: %w", err)
	}
	return nil
}

func (r *sourceRouteRepository) Delete(ctx context.Context, userID uuid.UUID, sourceType string, sourceKey uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM source_routes
		WHERE user_id = $1 AND source_type = $2 AND source_key = $3
	`, userID, sourceType, sourceKey)
	if err != nil {
		return fmt.Errorf("failed to delete source route: %w", err)
	}
	return nil
}

func (r *sourceRouteRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.SourceRoute, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, source_type, source_key, list
		FROM source_routes
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list source routes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var routes []*models.SourceRoute
	for rows.Next() {
		route := &models.SourceRoute{}
		if err := rows.Scan(&route.UserID, &route.SourceType, &route.SourceKey, &route.List); err != nil {
			return nil, fmt.Errorf("failed to scan source route: %w", err)
		}
		routes = append(routes, route)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating source route rows: %w", err)
	}
	return routes, nil
}
