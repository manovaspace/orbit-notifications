// Package domain holds the notification types and the ports every other layer
// depends on. It imports no infrastructure package.
package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Delivery status values stored in delivery_records.status.
const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"
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

// SendInput is an application-layer send request.
type SendInput struct {
	Template      string
	Channel       string
	Recipient     string
	Vars          map[string]string
	CorrelationID string
}

// HashRecipient returns the SHA-256 hex digest stored in delivery_records.
// The raw recipient must never reach the database outside dev payloads.
func HashRecipient(recipient string) string {
	sum := sha256.Sum256([]byte(recipient))
	return hex.EncodeToString(sum[:])
}

// Store persists delivery records.
type Store interface {
	Insert(ctx context.Context, record DeliveryRecord) error
	List(ctx context.Context, limit int, channel string) ([]DeliveryRecord, error)
}

// MailSender delivers email messages.
type MailSender interface {
	Send(ctx context.Context, to, subject, text, html string) error
}

// Flags evaluates boolean feature flags.
type Flags interface {
	Bool(ctx context.Context, flag string, defaultVal bool) bool
}

// Clock supplies the current time.
type Clock interface {
	Now() time.Time
}

// IDGenerator supplies delivery ids.
type IDGenerator interface {
	NewID() string
}
