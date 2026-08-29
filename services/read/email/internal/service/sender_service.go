package service

import (
	"context"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/repository"
	"github.com/google/uuid"
)

// SenderService handles sender tracking and management.
type SenderService interface {
	// UpsertOnReceipt creates or updates a sender record when an email is received.
	UpsertOnReceipt(ctx context.Context, userID uuid.UUID, senderEmail string, senderName string, receivedAt time.Time) (*models.EmailSender, error)

	// ListByUser returns senders for a user, paginated and ordered by last_received_at DESC.
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error)
}

type senderService struct {
	senderRepo repository.SenderRepository
}

// NewSenderService creates a new SenderService.
func NewSenderService(senderRepo repository.SenderRepository) SenderService {
	return &senderService{
		senderRepo: senderRepo,
	}
}

func (s *senderService) UpsertOnReceipt(ctx context.Context, userID uuid.UUID, senderEmail string, senderName string, receivedAt time.Time) (*models.EmailSender, error) {
	var name *string
	if senderName != "" {
		name = &senderName
	}

	sender := &models.EmailSender{
		UserID:         userID,
		SenderEmail:    senderEmail,
		SenderName:     name,
		EmailCount:     1,
		LastReceivedAt: &receivedAt,
	}

	if err := s.senderRepo.Upsert(ctx, sender); err != nil {
		return nil, fmt.Errorf("failed to upsert sender: %w", err)
	}

	return sender, nil
}

func (s *senderService) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
	senders, err := s.senderRepo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list senders: %w", err)
	}
	return senders, nil
}
