package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/manovaspace/orbit-notifications/internal/application/templates"
	"github.com/manovaspace/orbit-notifications/internal/domain"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/featureflags"
)

// Service orchestrates notification delivery.
type Service struct {
	repo   domain.DeliveryRepository
	mail   domain.MailSender
	flags  *featureflags.Evaluator
	isDev  bool
	logger func(msg string, args ...any)
}

func NewService(repo domain.DeliveryRepository, mail domain.MailSender, flags *featureflags.Evaluator, logger func(string, ...any)) *Service {
	return &Service{
		repo:   repo,
		mail:   mail,
		flags:  flags,
		isDev:  os.Getenv("DEPLOYMENT_ENVIRONMENT") == "dev",
		logger: logger,
	}
}

func (s *Service) Send(ctx context.Context, in domain.SendInput) (domain.DeliveryRecord, error) {
	if in.Recipient == "" || in.Template == "" || in.Channel == "" {
		return domain.DeliveryRecord{}, fmt.Errorf("send: template, channel, and recipient required")
	}

	subject, body, err := templates.Render(in.Template, in.Vars)
	if err != nil {
		return domain.DeliveryRecord{}, err
	}

	id := uuid.NewString()
	record := domain.DeliveryRecord{
		ID:            id,
		Template:      in.Template,
		Channel:       in.Channel,
		RecipientHash: hashRecipient(in.Recipient),
		Status:        "pending",
		CorrelationID: in.CorrelationID,
	}

	if s.isDev && s.devPayloadEnabled(ctx) {
		payload, _ := json.Marshal(map[string]string{
			"recipient": in.Recipient,
			"subject":   subject,
			"body":      body,
		})
		record.DevPayload = string(payload)
	}

	if in.Channel == "email" {
		if err := s.mail.Send(ctx, in.Recipient, subject, body); err != nil {
			record.Status = "failed"
			_ = s.repo.Insert(ctx, record)
			return record, err
		}
	} else if in.Channel == "sms" {
		// ponytail: dev records dev_payload only; prod needs SMSSender (not wired yet)
		if !s.isDev {
			record.Status = "failed"
			_ = s.repo.Insert(ctx, record)
			return record, fmt.Errorf("send: sms provider not configured")
		}
	} else {
		return domain.DeliveryRecord{}, fmt.Errorf("send: unsupported channel %q", in.Channel)
	}

	record.Status = "sent"
	if err := s.repo.Insert(ctx, record); err != nil {
		return record, err
	}

	s.logger("delivery_sent", "delivery.id", id, "channel", in.Channel, "template", in.Template)
	return record, nil
}

func (s *Service) ListDeliveries(ctx context.Context, limit int, channel string) ([]domain.DeliveryRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.List(ctx, limit, channel)
}

func hashRecipient(recipient string) string {
	sum := sha256.Sum256([]byte(recipient))
	return hex.EncodeToString(sum[:])
}

func (s *Service) devPayloadEnabled(ctx context.Context) bool {
	if s.flags == nil {
		return false
	}
	return s.flags.Bool(ctx, "manova.notifications.dev_payload", false)
}
