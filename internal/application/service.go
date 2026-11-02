package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/manovaspace/orbit-notifications/internal/application/templates"
	"github.com/manovaspace/orbit-notifications/internal/domain"
)

// devPayloadFlag persists plaintext OTP/payload in delivery_records.dev_payload
// when the environment is dev and the flag is on.
const devPayloadFlag = "manova.notifications.dev_payload"

// Deps are the application service's ports and runtime configuration. Every
// dependency is injected; the application layer reads no environment variable
// and imports no infrastructure package.
type Deps struct {
	Store  domain.Store
	Mail   domain.MailSender
	Flags  domain.Flags
	Clock  domain.Clock
	IDs    domain.IDGenerator
	Dev    bool
	Logger func(msg string, args ...any)
}

// Service orchestrates notification delivery.
type Service struct {
	store  domain.Store
	mail   domain.MailSender
	flags  domain.Flags
	clock  domain.Clock
	ids    domain.IDGenerator
	dev    bool
	logger func(msg string, args ...any)
}

// NewService builds the service from its ports.
func NewService(deps Deps) *Service {
	logger := deps.Logger
	if logger == nil {
		logger = func(string, ...any) {}
	}
	return &Service{
		store:  deps.Store,
		mail:   deps.Mail,
		flags:  deps.Flags,
		clock:  deps.Clock,
		ids:    deps.IDs,
		dev:    deps.Dev,
		logger: logger,
	}
}

// Send renders one template and delivers it on the requested channel, recording
// the outcome in delivery_records.
func (s *Service) Send(ctx context.Context, in domain.SendInput) (domain.DeliveryRecord, error) {
	if in.Recipient == "" || in.Template == "" || in.Channel == "" {
		return domain.DeliveryRecord{}, fmt.Errorf("send: template, channel, and recipient required")
	}

	res, err := templates.Render(in.Template, in.Vars)
	if err != nil {
		return domain.DeliveryRecord{}, err
	}

	id := s.ids.NewID()
	record := domain.DeliveryRecord{
		ID:            id,
		Template:      in.Template,
		Channel:       in.Channel,
		RecipientHash: domain.HashRecipient(in.Recipient),
		Status:        domain.StatusPending,
		CorrelationID: in.CorrelationID,
		CreatedAt:     s.clock.Now().UTC(),
	}

	if s.dev && s.devPayloadEnabled(ctx) {
		payload, _ := json.Marshal(map[string]string{
			"recipient": in.Recipient,
			"subject":   res.Subject,
			"body":      res.Text,
		})
		record.DevPayload = string(payload)
	}

	switch in.Channel {
	case "email":
		if err := s.mail.Send(ctx, in.Recipient, res.Subject, res.Text, res.HTML); err != nil {
			record.Status = domain.StatusFailed
			_ = s.store.Insert(ctx, record)
			return record, err
		}
	case "sms":
		// ponytail: dev records dev_payload only; prod needs SMSSender (not wired yet)
		if !s.dev {
			record.Status = domain.StatusFailed
			_ = s.store.Insert(ctx, record)
			return record, fmt.Errorf("send: sms provider not configured")
		}
	default:
		return domain.DeliveryRecord{}, fmt.Errorf("send: unsupported channel %q", in.Channel)
	}

	record.Status = domain.StatusSent
	if err := s.store.Insert(ctx, record); err != nil {
		return record, err
	}

	s.logger("delivery_sent", "delivery.id", id, "channel", in.Channel, "template", in.Template)
	return record, nil
}

// ListDeliveries returns the most recent deliveries, optionally filtered by
// channel. A limit outside 1..100 falls back to 50.
func (s *Service) ListDeliveries(ctx context.Context, limit int, channel string) ([]domain.DeliveryRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.List(ctx, limit, channel)
}

func (s *Service) devPayloadEnabled(ctx context.Context) bool {
	if s.flags == nil {
		return false
	}
	return s.flags.Bool(ctx, devPayloadFlag, false)
}
