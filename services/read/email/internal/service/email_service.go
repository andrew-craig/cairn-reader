package service

import (
	"context"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/repository"
)

// IngestEmailInput contains the data needed to ingest an email.
type IngestEmailInput struct {
	Recipient   string
	SenderEmail string
	SenderName  string
	Subject     string
	HTMLBody    string
	TextBody    string
	ReceivedAt  time.Time
}

// EmailService handles email ingestion orchestration.
type EmailService interface {
	// IngestEmail resolves the recipient, stores the raw email, and returns whether it was accepted.
	IngestEmail(ctx context.Context, input IngestEmailInput) (bool, error)
}

type emailService struct {
	addressService AddressService
	rawEmailRepo   repository.RawEmailRepository
}

// NewEmailService creates a new EmailService.
func NewEmailService(addressService AddressService, rawEmailRepo repository.RawEmailRepository) EmailService {
	return &emailService{
		addressService: addressService,
		rawEmailRepo:   rawEmailRepo,
	}
}

func (s *emailService) IngestEmail(ctx context.Context, input IngestEmailInput) (bool, error) {
	// Resolve recipient email address to user_id
	userID, err := s.addressService.ResolveRecipient(ctx, input.Recipient)
	if err == ErrRecipientNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to resolve recipient: %w", err)
	}

	// Build the raw email record
	rawEmail := &models.RawEmail{
		UserID:           userID,
		Recipient:        input.Recipient,
		SenderEmail:      input.SenderEmail,
		ReceivedAt:       input.ReceivedAt,
		ProcessingStatus: models.ProcessingStatusPending,
	}

	if input.SenderName != "" {
		rawEmail.SenderName = &input.SenderName
	}
	if input.Subject != "" {
		rawEmail.Subject = &input.Subject
	}
	if input.HTMLBody != "" {
		rawEmail.HTMLBody = &input.HTMLBody
	}
	if input.TextBody != "" {
		rawEmail.TextBody = &input.TextBody
	}

	// Store the raw email for later processing by the worker
	if err := s.rawEmailRepo.Create(ctx, rawEmail); err != nil {
		return false, fmt.Errorf("failed to store raw email: %w", err)
	}

	return true, nil
}
