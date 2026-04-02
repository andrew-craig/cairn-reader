package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAddressRepository is a test mock for repository.AddressRepository.
type mockAddressRepository struct {
	createFunc         func(ctx context.Context, address *models.EmailAddress) error
	getByUserIDFunc    func(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error)
	getByLocalPartFunc func(ctx context.Context, localPart string) (*models.EmailAddress, error)
	deleteFunc         func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockAddressRepository) Create(ctx context.Context, address *models.EmailAddress) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, address)
	}
	return nil
}

func (m *mockAddressRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockAddressRepository) GetByLocalPart(ctx context.Context, localPart string) (*models.EmailAddress, error) {
	if m.getByLocalPartFunc != nil {
		return m.getByLocalPartFunc(ctx, localPart)
	}
	return nil, nil
}

func (m *mockAddressRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userID)
	}
	return nil
}

func TestAddressService_GetOrCreate_ReturnsExisting(t *testing.T) {
	userID := uuid.New()
	existing := &models.EmailAddress{
		ID:        uuid.New(),
		UserID:    userID,
		LocalPart: "abc12345",
	}

	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, uid uuid.UUID) (*models.EmailAddress, error) {
			assert.Equal(t, userID, uid)
			return existing, nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetOrCreate(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, existing, result)
}

func TestAddressService_GetOrCreate_CreatesNew(t *testing.T) {
	userID := uuid.New()

	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, addr *models.EmailAddress) error {
			assert.Equal(t, userID, addr.UserID)
			assert.Len(t, addr.LocalPart, localPartLength)
			return nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetOrCreate(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Len(t, result.LocalPart, localPartLength)
}

func TestAddressService_GetOrCreate_CollisionRetry(t *testing.T) {
	userID := uuid.New()
	attempts := 0

	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, addr *models.EmailAddress) error {
			attempts++
			if attempts < 3 {
				return fmt.Errorf("failed to create email address: duplicate key value violates unique constraint")
			}
			return nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetOrCreate(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, attempts)
}

func TestAddressService_GetOrCreate_AllCollisionsExhausted(t *testing.T) {
	userID := uuid.New()

	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, _ *models.EmailAddress) error {
			return fmt.Errorf("failed to create email address: duplicate key value violates unique constraint")
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetOrCreate(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to generate unique local part")
}

func TestAddressService_GetOrCreate_NonCollisionError(t *testing.T) {
	userID := uuid.New()

	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, _ *models.EmailAddress) error {
			return fmt.Errorf("connection refused")
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetOrCreate(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestAddressService_GetOrCreate_GetByUserIDError(t *testing.T) {
	userID := uuid.New()

	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetOrCreate(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAddressService_GetByUserID(t *testing.T) {
	userID := uuid.New()
	expected := &models.EmailAddress{
		ID:        uuid.New(),
		UserID:    userID,
		LocalPart: "test1234",
	}

	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, uid uuid.UUID) (*models.EmailAddress, error) {
			assert.Equal(t, userID, uid)
			return expected, nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetByUserID(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestAddressService_GetByUserID_NotFound(t *testing.T) {
	repo := &mockAddressRepository{
		getByUserIDFunc: func(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
			return nil, nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.GetByUserID(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestAddressService_ResolveRecipient(t *testing.T) {
	userID := uuid.New()

	repo := &mockAddressRepository{
		getByLocalPartFunc: func(_ context.Context, localPart string) (*models.EmailAddress, error) {
			assert.Equal(t, "k7m2x9pq", localPart)
			return &models.EmailAddress{
				ID:        uuid.New(),
				UserID:    userID,
				LocalPart: localPart,
			}, nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.ResolveRecipient(context.Background(), "k7m2x9pq@read.cairnapp.com")

	require.NoError(t, err)
	assert.Equal(t, userID, result)
}

func TestAddressService_ResolveRecipient_WithoutDomain(t *testing.T) {
	userID := uuid.New()

	repo := &mockAddressRepository{
		getByLocalPartFunc: func(_ context.Context, localPart string) (*models.EmailAddress, error) {
			assert.Equal(t, "k7m2x9pq", localPart)
			return &models.EmailAddress{
				ID:        uuid.New(),
				UserID:    userID,
				LocalPart: localPart,
			}, nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.ResolveRecipient(context.Background(), "k7m2x9pq")

	require.NoError(t, err)
	assert.Equal(t, userID, result)
}

func TestAddressService_ResolveRecipient_NotFound(t *testing.T) {
	repo := &mockAddressRepository{
		getByLocalPartFunc: func(_ context.Context, _ string) (*models.EmailAddress, error) {
			return nil, nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.ResolveRecipient(context.Background(), "unknown@example.com")

	assert.ErrorIs(t, err, ErrRecipientNotFound)
	assert.Equal(t, uuid.Nil, result)
}

func TestAddressService_ResolveRecipient_EmptyRecipient(t *testing.T) {
	repo := &mockAddressRepository{}

	svc := NewAddressService(repo)
	result, err := svc.ResolveRecipient(context.Background(), "")

	assert.ErrorIs(t, err, ErrRecipientNotFound)
	assert.Equal(t, uuid.Nil, result)
}

func TestAddressService_ResolveRecipient_CaseInsensitive(t *testing.T) {
	userID := uuid.New()

	repo := &mockAddressRepository{
		getByLocalPartFunc: func(_ context.Context, localPart string) (*models.EmailAddress, error) {
			assert.Equal(t, "abc12345", localPart)
			return &models.EmailAddress{
				ID:        uuid.New(),
				UserID:    userID,
				LocalPart: localPart,
			}, nil
		},
	}

	svc := NewAddressService(repo)
	result, err := svc.ResolveRecipient(context.Background(), "ABC12345@example.com")

	require.NoError(t, err)
	assert.Equal(t, userID, result)
}

func TestExtractLocalPart(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full email", "k7m2x9pq@read.cairnapp.com", "k7m2x9pq"},
		{"local only", "k7m2x9pq", "k7m2x9pq"},
		{"uppercase", "ABC123@EXAMPLE.COM", "abc123"},
		{"with whitespace", "  test@example.com  ", "test"},
		{"empty", "", ""},
		{"at sign only", "@example.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLocalPart(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateLocalPart(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		part, err := generateLocalPart()
		require.NoError(t, err)
		assert.Len(t, part, localPartLength)

		// Verify all characters are in the charset
		for _, c := range part {
			assert.Contains(t, localPartCharset, string(c))
		}

		seen[part] = true
	}

	// With 36^8 possible values, 100 random generations should all be unique
	assert.Equal(t, 100, len(seen), "expected all generated local parts to be unique")
}
