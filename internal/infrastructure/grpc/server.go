package grpc

import (
	"context"

	notificationsv1 "github.com/manovaspace/orbit-notifications/api/notifications/v1"
	"github.com/manovaspace/orbit-notifications/internal/application"
	"github.com/manovaspace/orbit-notifications/internal/domain"
)

// Server implements NotificationsService gRPC API.
type Server struct {
	notificationsv1.UnimplementedNotificationsServiceServer
	svc *application.Service
}

func New(svc *application.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) Send(ctx context.Context, req *notificationsv1.SendRequest) (*notificationsv1.SendResponse, error) {
	rec, err := s.svc.Send(ctx, domain.SendInput{
		Template:      req.GetTemplate(),
		Channel:       req.GetChannel(),
		Recipient:     req.GetRecipient(),
		Vars:          req.GetVars(),
		CorrelationID: req.GetCorrelationId(),
	})
	if err != nil {
		return nil, err
	}
	return &notificationsv1.SendResponse{
		DeliveryId: rec.ID,
		Status:     rec.Status,
	}, nil
}

func (s *Server) ListDeliveries(ctx context.Context, req *notificationsv1.ListDeliveriesRequest) (*notificationsv1.ListDeliveriesResponse, error) {
	recs, err := s.svc.ListDeliveries(ctx, int(req.GetLimit()), req.GetChannel())
	if err != nil {
		return nil, err
	}
	out := make([]*notificationsv1.DeliveryRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, &notificationsv1.DeliveryRecord{
			Id:            r.ID,
			Template:      r.Template,
			Channel:       r.Channel,
			Status:        r.Status,
			CorrelationId: r.CorrelationID,
			CreatedAt:     r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			DevPayload:    r.DevPayload,
		})
	}
	return &notificationsv1.ListDeliveriesResponse{Deliveries: out}, nil
}
