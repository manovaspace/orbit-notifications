package domain

import (
	"context"
	"time"
)

// DeliveryRecord is a persisted notification delivery.
type DeliveryRecord struct {
	ID            string
	Template      string
	Channel       string
	RecipientHash string
	Status        string
	DevPayload    string
	CorrelationID string
	CreatedAt     time.Time
}

// SendInput is application-layer send request.
type SendInput struct {
	Template      string
	Channel       string
	Recipient     string
	Vars          map[string]string
	CorrelationID string
}

// DeliveryRepository persists delivery records.
type DeliveryRepository interface {
	Insert(ctx context.Context, record DeliveryRecord) error
	List(ctx context.Context, limit int, channel string) ([]DeliveryRecord, error)
}

// MailSender delivers email messages.
type MailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}
