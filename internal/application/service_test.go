package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/manovaspace/orbit-notifications/internal/domain"
)

type fakeStore struct {
	inserted   []domain.DeliveryRecord
	insertErr  error
	listed     []domain.DeliveryRecord
	listErr    error
	listLimit  int
	listChan   string
	insertCall int
}

func (s *fakeStore) Insert(_ context.Context, record domain.DeliveryRecord) error {
	s.insertCall++
	s.inserted = append(s.inserted, record)
	return s.insertErr
}

func (s *fakeStore) List(_ context.Context, limit int, channel string) ([]domain.DeliveryRecord, error) {
	s.listLimit = limit
	s.listChan = channel
	return s.listed, s.listErr
}

type fakeMail struct {
	calls   int
	err     error
	to      string
	subject string
	text    string
	html    string
}

func (m *fakeMail) Send(_ context.Context, to, subject, text, html string) error {
	m.calls++
	m.to, m.subject, m.text, m.html = to, subject, text, html
	return m.err
}

type fakeFlags map[string]bool

func (f fakeFlags) Bool(_ context.Context, flag string, defaultVal bool) bool {
	if v, ok := f[flag]; ok {
		return v
	}
	return defaultVal
}

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

type fixedIDs struct{ id string }

func (i fixedIDs) NewID() string { return i.id }

func testService(t *testing.T) (*Service, *fakeStore, *fakeMail) {
	t.Helper()
	store := &fakeStore{}
	mail := &fakeMail{}
	return NewService(Deps{
		Store: store,
		Mail:  mail,
		Flags: fakeFlags{},
		Clock: fixedClock{at: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)},
		IDs:   fixedIDs{id: "11111111-1111-1111-1111-111111111111"},
		Dev:   false,
	}), store, mail
}

func validInput() domain.SendInput {
	return domain.SendInput{
		Template:      "otp_login",
		Channel:       "email",
		Recipient:     "person@example.com",
		Vars:          map[string]string{"code": "123456"},
		CorrelationID: "corr-1",
	}
}

func TestSendEmailRecordsSentDelivery(t *testing.T) {
	svc, store, mail := testService(t)

	rec, err := svc.Send(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	if mail.calls != 1 || mail.to != "person@example.com" || mail.subject != "Your Manova login code" {
		t.Fatalf("mail call = %d to=%q subject=%q", mail.calls, mail.to, mail.subject)
	}
	if !strings.Contains(mail.text, "123456") || mail.html == "" {
		t.Fatalf("rendered mail missing code or html: text=%q html=%d bytes", mail.text, len(mail.html))
	}
	if len(store.inserted) != 1 {
		t.Fatalf("inserted %d records, want 1", len(store.inserted))
	}
	got := store.inserted[0]
	want := domain.DeliveryRecord{
		ID:            "11111111-1111-1111-1111-111111111111",
		Template:      "otp_login",
		Channel:       "email",
		RecipientHash: domain.HashRecipient("person@example.com"),
		Status:        domain.StatusSent,
		CorrelationID: "corr-1",
		CreatedAt:     time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC),
	}
	if got != want {
		t.Fatalf("inserted record = %+v, want %+v", got, want)
	}
	if rec.Status != domain.StatusSent || rec.ID != want.ID {
		t.Fatalf("returned record = %+v", rec)
	}
	if strings.Contains(rec.RecipientHash, "person@example.com") {
		t.Fatal("recipient hash contains the raw recipient")
	}
}

func TestSendEmailFailureRecordsFailedDelivery(t *testing.T) {
	svc, store, mail := testService(t)
	mail.err = errors.New("smtp down")

	rec, err := svc.Send(context.Background(), validInput())
	if err == nil || !strings.Contains(err.Error(), "smtp down") {
		t.Fatalf("err = %v", err)
	}
	if len(store.inserted) != 1 || store.inserted[0].Status != domain.StatusFailed {
		t.Fatalf("inserted = %+v", store.inserted)
	}
	if rec.Status != domain.StatusFailed {
		t.Fatalf("status = %q", rec.Status)
	}
}

func TestSendSMSSucceedsOnlyInDev(t *testing.T) {
	svc, store, mail := testService(t)
	input := validInput()
	input.Channel = "sms"
	input.Template = "otp_login_sms"

	if _, err := svc.Send(context.Background(), input); err == nil || !strings.Contains(err.Error(), "sms provider not configured") {
		t.Fatalf("prod sms err = %v", err)
	}
	if mail.calls != 0 {
		t.Fatalf("sms used the mail sender %d times", mail.calls)
	}
	if len(store.inserted) != 1 || store.inserted[0].Status != domain.StatusFailed {
		t.Fatalf("prod sms insert = %+v", store.inserted)
	}

	store.inserted = nil
	devSvc := NewService(Deps{
		Store: store,
		Mail:  mail,
		Flags: fakeFlags{},
		Clock: fixedClock{at: time.Now()},
		IDs:   fixedIDs{id: "22222222-2222-2222-2222-222222222222"},
		Dev:   true,
	})
	if _, err := devSvc.Send(context.Background(), input); err != nil {
		t.Fatalf("dev sms err = %v", err)
	}
	if len(store.inserted) != 1 || store.inserted[0].Status != domain.StatusSent {
		t.Fatalf("dev sms insert = %+v", store.inserted)
	}
	if mail.calls != 0 {
		t.Fatalf("sms used the mail sender %d times", mail.calls)
	}
}

func TestSendRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		edit func(*domain.SendInput)
		want string
	}{
		{"missing template", func(in *domain.SendInput) { in.Template = "" }, "template, channel, and recipient required"},
		{"missing channel", func(in *domain.SendInput) { in.Channel = "" }, "template, channel, and recipient required"},
		{"missing recipient", func(in *domain.SendInput) { in.Recipient = "" }, "template, channel, and recipient required"},
		{"unknown channel", func(in *domain.SendInput) { in.Channel = "push" }, `unsupported channel "push"`},
		{"template render", func(in *domain.SendInput) { in.Vars = nil }, "requires code"},
		{"unknown template", func(in *domain.SendInput) { in.Template = "nope" }, `unknown template "nope"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, store, mail := testService(t)
			in := validInput()
			tc.edit(&in)

			_, err := svc.Send(context.Background(), in)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if store.insertCall != 0 || mail.calls != 0 {
				t.Fatalf("invalid input reached a port: store=%d mail=%d", store.insertCall, mail.calls)
			}
		})
	}
}

func TestSendDevPayloadNeedsDevAndFlag(t *testing.T) {
	tests := []struct {
		name    string
		dev     bool
		flag    bool
		wantRaw bool
	}{
		{"dev with flag", true, true, true},
		{"dev without flag", true, false, false},
		{"prod with flag", false, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{}
			svc := NewService(Deps{
				Store: store,
				Mail:  &fakeMail{},
				Flags: fakeFlags{devPayloadFlag: tc.flag},
				Clock: fixedClock{at: time.Now()},
				IDs:   fixedIDs{id: "33333333-3333-3333-3333-333333333333"},
				Dev:   tc.dev,
			})

			if _, err := svc.Send(context.Background(), validInput()); err != nil {
				t.Fatal(err)
			}
			payload := store.inserted[0].DevPayload
			if tc.wantRaw {
				for _, want := range []string{`"recipient":"person@example.com"`, `"subject":"Your Manova login code"`, `123456`} {
					if !strings.Contains(payload, want) {
						t.Fatalf("dev payload %q missing %q", payload, want)
					}
				}
			} else if payload != "" {
				t.Fatalf("dev payload = %q, want empty", payload)
			}
		})
	}
}

func TestListDeliveriesClampsLimitAndPassesChannel(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{-1, 50}, {0, 50}, {1, 1}, {100, 100}, {101, 50},
	}
	for _, tc := range tests {
		store := &fakeStore{}
		svc := NewService(Deps{
			Store: store,
			Mail:  &fakeMail{},
			Flags: fakeFlags{},
			Clock: fixedClock{at: time.Now()},
			IDs:   fixedIDs{id: "id"},
		})
		if _, err := svc.ListDeliveries(context.Background(), tc.in, "email"); err != nil {
			t.Fatal(err)
		}
		if store.listLimit != tc.want || store.listChan != "email" {
			t.Fatalf("in=%d -> limit=%d channel=%q, want %d/email", tc.in, store.listLimit, store.listChan, tc.want)
		}
	}
}

func TestListDeliveriesReturnsStoreError(t *testing.T) {
	store := &fakeStore{listErr: errors.New("db down")}
	svc := NewService(Deps{
		Store: store,
		Mail:  &fakeMail{},
		Flags: fakeFlags{},
		Clock: fixedClock{at: time.Now()},
		IDs:   fixedIDs{id: "id"},
	})
	if _, err := svc.ListDeliveries(context.Background(), 10, ""); err == nil {
		t.Fatal("store error was swallowed")
	}
}
