package handlers

import (
	"context"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockSourceRouteRepository is a mock implementation of repository.SourceRouteRepository
type MockSourceRouteRepository struct {
	mock.Mock
}

func (m *MockSourceRouteRepository) Upsert(ctx context.Context, route *models.SourceRoute) error {
	args := m.Called(ctx, route)
	return args.Error(0)
}

func (m *MockSourceRouteRepository) Delete(ctx context.Context, userID uuid.UUID, sourceType string, sourceKey uuid.UUID) error {
	args := m.Called(ctx, userID, sourceType, sourceKey)
	return args.Error(0)
}

func (m *MockSourceRouteRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.SourceRoute, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SourceRoute), args.Error(1)
}
