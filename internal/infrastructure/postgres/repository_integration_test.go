package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/manovaspace/orbit-notifications/internal/domain"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/postgres/migrations"
	"github.com/manovaspace/orbit-notifications/internal/testutil"
)

func migrationPool(t *testing.T) (*Repository, *testutil.Database) {
	t.Helper()
	db := testutil.OpenPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := migrations.Open(db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()
	if _, err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	return NewRepository(db.Pool), db
}

func TestRepositoryInsertAndList(t *testing.T) {
	repo, _ := migrationPool(t)
	ctx := context.Background()

	older := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	records := []domain.DeliveryRecord{
		{
			ID:            "11111111-1111-1111-1111-111111111111",
			Template:      "otp_login",
			Channel:       "email",
			RecipientHash: domain.HashRecipient("a@example.com"),
			Status:        domain.StatusSent,
			CorrelationID: "corr-old",
			CreatedAt:     older,
		},
		{
			ID:            "22222222-2222-2222-2222-222222222222",
			Template:      "otp_login_sms",
			Channel:       "sms",
			RecipientHash: domain.HashRecipient("b@example.com"),
			Status:        domain.StatusFailed,
			DevPayload:    `{"recipient":"b@example.com"}`,
			CorrelationID: "",
			CreatedAt:     newer,
		},
	}
	for _, record := range records {
		if err := repo.Insert(ctx, record); err != nil {
			t.Fatalf("insert %s: %v", record.ID, err)
		}
	}

	got, err := repo.List(ctx, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("listed %d records, want 2", len(got))
	}
	if got[0].ID != records[1].ID || got[1].ID != records[0].ID {
		t.Fatalf("order = %s,%s; want newest first", got[0].ID, got[1].ID)
	}
	if got[1].DevPayload != "" || got[1].CorrelationID != "corr-old" || !got[1].CreatedAt.Equal(older) {
		t.Fatalf("email record = %+v", got[1])
	}
	if got[0].DevPayload != `{"recipient": "b@example.com"}` || got[0].Status != domain.StatusFailed {
		t.Fatalf("sms record = %+v", got[0])
	}

	emailOnly, err := repo.List(ctx, 10, "email")
	if err != nil {
		t.Fatal(err)
	}
	if len(emailOnly) != 1 || emailOnly[0].ID != records[0].ID {
		t.Fatalf("email filter = %+v", emailOnly)
	}

	limited, err := repo.List(ctx, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 || limited[0].ID != records[1].ID {
		t.Fatalf("limit = %+v", limited)
	}
}

func TestRepositoryZeroCreatedAtUsesDatabaseDefault(t *testing.T) {
	repo, _ := migrationPool(t)
	ctx := context.Background()

	before := time.Now().Add(-time.Minute)
	err := repo.Insert(ctx, domain.DeliveryRecord{
		ID:            "33333333-3333-3333-3333-333333333333",
		Template:      "otp_login",
		Channel:       "email",
		RecipientHash: "hash",
		Status:        domain.StatusSent,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.List(ctx, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CreatedAt.Before(before) {
		t.Fatalf("created_at = %v, want a database default after %v", got[0].CreatedAt, before)
	}
}

func TestRepositoryListEmptyIsNotNil(t *testing.T) {
	repo, _ := migrationPool(t)
	got, err := repo.List(context.Background(), 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("empty list must be an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("listed %d records", len(got))
	}
}

func TestRepositoryRejectsInvalidID(t *testing.T) {
	repo, _ := migrationPool(t)
	err := repo.Insert(context.Background(), domain.DeliveryRecord{ID: "not-a-uuid"})
	if err == nil {
		t.Fatal("invalid id was accepted")
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: %v", err)
	}
}
