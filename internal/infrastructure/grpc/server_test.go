package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	notificationsv1 "github.com/manovaspace/orbit-notifications/api/notifications/v1"
	"github.com/manovaspace/orbit-notifications/internal/domain"
)

type fakeNotifier struct {
	sendInput domain.SendInput
	sendRec   domain.DeliveryRecord
	sendErr   error

	listLimit   int
	listChannel string
	listRecs    []domain.DeliveryRecord
	listErr     error
}

func (f *fakeNotifier) Send(_ context.Context, in domain.SendInput) (domain.DeliveryRecord, error) {
	f.sendInput = in
	return f.sendRec, f.sendErr
}

func (f *fakeNotifier) ListDeliveries(_ context.Context, limit int, channel string) ([]domain.DeliveryRecord, error) {
	f.listLimit = limit
	f.listChannel = channel
	return f.listRecs, f.listErr
}

func TestSendMapsRequestAndResponse(t *testing.T) {
	app := &fakeNotifier{sendRec: domain.DeliveryRecord{ID: "delivery-1", Status: domain.StatusSent}}
	srv := New(app)

	resp, err := srv.Send(context.Background(), &notificationsv1.SendRequest{
		Template:      "otp_login",
		Channel:       "email",
		Recipient:     "person@example.com",
		Vars:          map[string]string{"code": "123456"},
		CorrelationId: "corr-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantInput := domain.SendInput{
		Template:      "otp_login",
		Channel:       "email",
		Recipient:     "person@example.com",
		Vars:          map[string]string{"code": "123456"},
		CorrelationID: "corr-1",
	}
	if app.sendInput.Template != wantInput.Template || app.sendInput.Channel != wantInput.Channel ||
		app.sendInput.Recipient != wantInput.Recipient || app.sendInput.CorrelationID != wantInput.CorrelationID ||
		app.sendInput.Vars["code"] != "123456" {
		t.Fatalf("input = %+v, want %+v", app.sendInput, wantInput)
	}
	if resp.GetDeliveryId() != "delivery-1" || resp.GetStatus() != domain.StatusSent {
		t.Fatalf("response = %+v", resp)
	}
}

func TestSendReturnsApplicationError(t *testing.T) {
	srv := New(&fakeNotifier{sendErr: errors.New("smtp down")})
	if _, err := srv.Send(context.Background(), &notificationsv1.SendRequest{}); err == nil {
		t.Fatal("application error was swallowed")
	}
}

func TestListDeliveriesMapsRecords(t *testing.T) {
	app := &fakeNotifier{listRecs: []domain.DeliveryRecord{{
		ID:            "delivery-1",
		Template:      "otp_login",
		Channel:       "email",
		Status:        domain.StatusSent,
		DevPayload:    `{"code":"123456"}`,
		CorrelationID: "corr-1",
		CreatedAt:     time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC),
	}}}
	srv := New(app)

	resp, err := srv.ListDeliveries(context.Background(), &notificationsv1.ListDeliveriesRequest{Limit: 25, Channel: "email"})
	if err != nil {
		t.Fatal(err)
	}
	if app.listLimit != 25 || app.listChannel != "email" {
		t.Fatalf("limit=%d channel=%q", app.listLimit, app.listChannel)
	}
	if len(resp.GetDeliveries()) != 1 {
		t.Fatalf("deliveries = %d", len(resp.GetDeliveries()))
	}
	got := resp.GetDeliveries()[0]
	if got.GetId() != "delivery-1" || got.GetCreatedAt() != "2026-11-02T12:00:00Z" ||
		got.GetCorrelationId() != "corr-1" || got.GetDevPayload() != `{"code":"123456"}` {
		t.Fatalf("record = %+v", got)
	}
}

func TestListDeliveriesEmptyIsNotNil(t *testing.T) {
	srv := New(&fakeNotifier{})
	resp, err := srv.ListDeliveries(context.Background(), &notificationsv1.ListDeliveriesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetDeliveries() == nil {
		t.Fatal("empty list must stay an empty slice, as before")
	}
}
