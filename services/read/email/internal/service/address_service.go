package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/repository"
	"github.com/google/uuid"
)

const (
	localPartLength  = 8
	localPartCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
	maxRetries       = 5
)

// ErrRecipientNotFound is returned when a recipient email address cannot be resolved to a user.
var ErrRecipientNotFound = fmt.Errorf("recipient not found")

// AddressService handles email address generation and lookup logic.
type AddressService interface {
	// GetOrCreate returns the user's existing email address, or creates a new one.
	GetOrCreate(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error)

	// GetByUserID returns the user's email address, or nil if none exists.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error)

	// ResolveRecipient parses the local part from an email address and resolves it to a user ID.
	ResolveRecipient(ctx context.Context, recipient string) (uuid.UUID, error)
}

type addressService struct {
	addressRepo   repository.AddressRepository
	generateLocal func() (string, error)
}

// NewAddressService creates a new AddressService.
func NewAddressService(addressRepo repository.AddressRepository) AddressService {
	return &addressService{
		addressRepo:   addressRepo,
		generateLocal: generateLocalPart,
	}
}

func (s *addressService) GetOrCreate(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error) {
	existing, err := s.addressRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing address: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		localPart, err := s.generateLocal()
		if err != nil {
			return nil, fmt.Errorf("failed to generate local part: %w", err)
		}

		address := &models.EmailAddress{
			UserID:    userID,
			LocalPart: localPart,
		}

		err = s.addressRepo.Create(ctx, address)
		if err != nil {
			// On collision (duplicate local_part), retry with a new random string.
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				continue
			}
			return nil, fmt.Errorf("failed to create email address: %w", err)
		}

		return address, nil
	}

	return nil, fmt.Errorf("failed to generate unique local part after %d attempts", maxRetries)
}

func (s *addressService) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error) {
	address, err := s.addressRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email address: %w", err)
	}
	return address, nil
}

func (s *addressService) ResolveRecipient(ctx context.Context, recipient string) (uuid.UUID, error) {
	localPart := extractLocalPart(recipient)
	if localPart == "" {
		return uuid.Nil, ErrRecipientNotFound
	}

	address, err := s.addressRepo.GetByLocalPart(ctx, localPart)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to resolve recipient: %w", err)
	}
	if address == nil {
		return uuid.Nil, ErrRecipientNotFound
	}

	return address.UserID, nil
}

// extractLocalPart extracts the local part from an email address.
// For "k7m2x9pq@anything.com" it returns "k7m2x9pq".
// For just "k7m2x9pq" it returns "k7m2x9pq".
func extractLocalPart(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	idx := strings.Index(email, "@")
	if idx >= 0 {
		local := email[:idx]
		if local == "" {
			return ""
		}
		return strings.ToLower(local)
	}
	return strings.ToLower(email)
}

// generateLocalPart generates a cryptographically random 8-character lowercase alphanumeric string.
func generateLocalPart() (string, error) {
	charsetLen := big.NewInt(int64(len(localPartCharset)))
	b := make([]byte, localPartLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		b[i] = localPartCharset[n.Int64()]
	}
	return string(b), nil
}
