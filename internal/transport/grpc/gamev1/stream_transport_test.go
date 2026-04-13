package gamev1

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type streamTransportService struct {
	panicTransportService
	subscribeResult service.EventStream
	subscribeErr    error
	subscribe       func(context.Context, caller.Caller, string, uint64) (service.EventStream, error)
}

func (s *streamTransportService) SubscribeEvents(ctx context.Context, act caller.Caller, campaignID string, afterSeq uint64) (service.EventStream, error) {
	if s.subscribe != nil {
		return s.subscribe(ctx, act, campaignID, afterSeq)
	}
	return s.subscribeResult, s.subscribeErr
}

func TestStreamCampaignEvents(t *testing.T) {
	t.Parallel()

	server := mustServer(t, &panicTransportService{})
	if err := (*Server)(nil).StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{}, &fakeStream{}); status.Code(err) != codes.Internal {
		t.Fatalf("StreamCampaignEvents(nil server) code = %v, want %v", status.Code(err), codes.Internal)
	}
	if err := server.StreamCampaignEvents(nil, &fakeStream{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("StreamCampaignEvents(nil request) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if err := server.StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{}, nil); status.Code(err) != codes.Internal {
		t.Fatalf("StreamCampaignEvents(nil stream) code = %v, want %v", status.Code(err), codes.Internal)
	}

	ctx, cancel := context.WithCancel(inboundContext("subject-1"))
	stream := &fakeStream{ctx: ctx}
	svc := &streamTransportService{
		subscribe: func(context.Context, caller.Caller, string, uint64) (service.EventStream, error) {
			ch := make(chan event.Record, 1)
			ch <- event.Record{
				Seq:        1,
				CommitSeq:  1,
				RecordedAt: time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC),
				Envelope:   mustTransportEnvelope(t, "camp-1", campaign.PlayBegan{SessionID: "sess-1", SceneID: "scene-1"}),
			}
			close(ch)
			cancel()
			return service.EventStream{Records: ch, Close: func() {}}, nil
		},
	}
	server = mustServer(t, svc)
	if err := server.StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{CampaignId: "camp-1"}, stream); err != nil {
		t.Fatalf("StreamCampaignEvents() error = %v", err)
	}
	if got, want := len(stream.sent), 1; got != want {
		t.Fatalf("sent len = %d, want %d", got, want)
	}

	denyServer := mustServer(t, &streamTransportService{
		subscribeErr: &authz.DeniedError{Capability: authz.CapabilityReadCampaign, Reason: "nope"},
	})
	if err := denyServer.StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{CampaignId: "camp-1"}, &fakeStream{ctx: inboundContext("subject-1")}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("StreamCampaignEvents(denied) code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
	invalidServer := mustServer(t, &streamTransportService{
		subscribeErr: errs.InvalidArgumentf("campaign id is required"),
	})
	if err := invalidServer.StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{CampaignId: ""}, &fakeStream{ctx: inboundContext("subject-1")}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("StreamCampaignEvents(invalid campaign id) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if err := server.StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{CampaignId: "camp-1"}, &fakeStream{ctx: context.Background()}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("StreamCampaignEvents(unauthenticated) code = %v, want %v", status.Code(err), codes.Unauthenticated)
	}

	sendErrServer := mustServer(t, &streamTransportService{
		subscribe: func(context.Context, caller.Caller, string, uint64) (service.EventStream, error) {
			ch := make(chan event.Record, 1)
			ch <- event.Record{Envelope: mustTransportEnvelope(t, "camp-1", campaign.Created{Name: "Autumn Twilight"})}
			close(ch)
			return service.EventStream{Records: ch, Close: func() {}}, nil
		},
	})
	if err := sendErrServer.StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{CampaignId: "camp-1"}, &fakeStream{ctx: inboundContext("subject-1"), sendErr: io.EOF}); !errors.Is(err, io.EOF) {
		t.Fatalf("StreamCampaignEvents(send err) error = %v, want EOF", err)
	}

	badEventServer := mustServer(t, &streamTransportService{
		subscribe: func(context.Context, caller.Caller, string, uint64) (service.EventStream, error) {
			ch := make(chan event.Record, 1)
			ch <- event.Record{Envelope: event.Envelope{CampaignID: "camp-1", Message: unknownEvent{}}}
			close(ch)
			return service.EventStream{Records: ch, Close: func() {}}, nil
		},
	})
	if err := badEventServer.StreamCampaignEvents(&gamev1pb.StreamCampaignEventsRequest{CampaignId: "camp-1"}, &fakeStream{ctx: inboundContext("subject-1")}); status.Code(err) != codes.Internal {
		t.Fatalf("StreamCampaignEvents(bad event) code = %v, want %v", status.Code(err), codes.Internal)
	}
}
