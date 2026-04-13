package gamev1

import (
	"context"
	"fmt"
	"maps"
	"testing"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
	memorystorage "github.com/fracturing-space/game/internal/storage/memory"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func mustServer(t *testing.T, svc service.API) *Server {
	t.Helper()
	server, err := NewServer(svc)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func mustRealService(t *testing.T) *service.Service {
	t.Helper()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	stores := memorystorage.NewBundle()
	svc, err := service.New(service.Config{
		Manifest:        manifest,
		IDs:             newSequentialIDAllocator(),
		Journal:         stores.Journal,
		ProjectionStore: stores.ProjectionStore,
		ArtifactStore:   stores.ArtifactStore,
	})
	if err != nil {
		t.Fatalf("service.New() error = %v", err)
	}
	return svc
}

type sequentialIDAllocator struct {
	next map[string]uint64
}

func newSequentialIDAllocator() *sequentialIDAllocator {
	return &sequentialIDAllocator{next: make(map[string]uint64)}
}

func (a *sequentialIDAllocator) Session(commit bool) service.IDSession {
	next := make(map[string]uint64, len(a.next))
	maps.Copy(next, a.next)
	return &sequentialIDSession{
		allocator: a,
		next:      next,
		commit:    commit,
	}
}

type sequentialIDSession struct {
	allocator *sequentialIDAllocator
	next      map[string]uint64
	commit    bool
}

func (s *sequentialIDSession) NewID(prefix string) (string, error) {
	if prefix == "" {
		return "", fmt.Errorf("id prefix is required")
	}
	s.next[prefix]++
	return fmt.Sprintf("%s-%d", prefix, s.next[prefix]), nil
}

func (s *sequentialIDSession) Commit() {
	if s == nil || !s.commit || s.allocator == nil {
		return
	}
	s.allocator.next = s.next
}

func inboundContext(subjectID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(subjectIDHeader, subjectID))
}

func TestRequestCampaignID(t *testing.T) {
	t.Parallel()

	if _, err := requestCampaignID(""); err == nil {
		t.Fatal("requestCampaignID(blank) error = nil, want failure")
	}
	if _, err := requestCampaignID(" camp-1 "); err == nil {
		t.Fatal("requestCampaignID(padded) error = nil, want failure")
	}
	if got, err := requestCampaignID("camp-1"); err != nil || got != "camp-1" {
		t.Fatalf("requestCampaignID(valid) = (%q, %v), want camp-1,nil", got, err)
	}
}

func mustValidateTransportCommand(t *testing.T, envelope command.Envelope) command.Envelope {
	t.Helper()

	catalog, err := command.NewCatalog(
		campaign.CreateCommandSpec,
		campaign.UpdateCommandSpec,
		campaign.PlayBeginCommandSpec,
		character.CreateCommandSpec,
		character.UpdateCommandSpec,
		character.DeleteCommandSpec,
		participant.JoinCommandSpec,
		participant.UpdateCommandSpec,
		participant.BindCommandSpec,
		participant.UnbindCommandSpec,
		participant.LeaveCommandSpec,
		session.StartCommandSpec,
		session.EndCommandSpec,
		scene.CreateCommandSpec,
		scene.ActivateCommandSpec,
		scene.EndCommandSpec,
		scene.ReplaceCastCommandSpec,
		campaign.AIBindCommandSpec,
		campaign.AIUnbindCommandSpec,
	)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	validated, _, err := catalog.Validate(envelope)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	return validated
}

func mustTransportEnvelope(t *testing.T, campaignID string, message event.Message) event.Envelope {
	t.Helper()

	switch typed := message.(type) {
	case campaign.Created:
		envelope, err := event.NewEnvelope(campaign.CreatedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(created) error = %v", err)
		}
		return envelope
	case campaign.PlayBegan:
		envelope, err := event.NewEnvelope(campaign.PlayBeganEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(play began) error = %v", err)
		}
		return envelope
	case campaign.PlayPaused:
		envelope, err := event.NewEnvelope(campaign.PlayPausedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(play paused) error = %v", err)
		}
		return envelope
	case campaign.PlayResumed:
		envelope, err := event.NewEnvelope(campaign.PlayResumedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(play resumed) error = %v", err)
		}
		return envelope
	case campaign.PlayEnded:
		envelope, err := event.NewEnvelope(campaign.PlayEndedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(play ended) error = %v", err)
		}
		return envelope
	case campaign.AIBound:
		envelope, err := event.NewEnvelope(campaign.AIBoundEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(ai bound) error = %v", err)
		}
		return envelope
	case campaign.AIUnbound:
		envelope, err := event.NewEnvelope(campaign.AIUnboundEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(ai unbound) error = %v", err)
		}
		return envelope
	case character.Created:
		envelope, err := event.NewEnvelope(character.CreatedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(character created) error = %v", err)
		}
		return envelope
	case session.Started:
		envelope, err := event.NewEnvelope(session.StartedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(session started) error = %v", err)
		}
		return envelope
	case session.Ended:
		envelope, err := event.NewEnvelope(session.EndedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(session ended) error = %v", err)
		}
		return envelope
	case participant.Joined:
		envelope, err := event.NewEnvelope(participant.JoinedEventSpec, campaignID, typed)
		if err != nil {
			t.Fatalf("NewEnvelope(joined) error = %v", err)
		}
		return envelope
	default:
		return event.Envelope{CampaignID: campaignID, Message: message}
	}
}

func assertPlayReadinessStatusDetails(t *testing.T, err error, wantCode, wantMessage string) {
	t.Helper()

	st := status.Convert(err)
	if got, want := st.Code(), codes.FailedPrecondition; got != want {
		t.Fatalf("status code = %v, want %v", got, want)
	}
	if got := st.Message(); got != wantMessage {
		t.Fatalf("status message = %q, want %q", got, wantMessage)
	}

	var gotErrorInfo *errdetails.ErrorInfo
	var gotPrecondition *errdetails.PreconditionFailure
	for _, detail := range st.Details() {
		switch typed := detail.(type) {
		case *errdetails.ErrorInfo:
			gotErrorInfo = typed
		case *errdetails.PreconditionFailure:
			gotPrecondition = typed
		}
	}
	if gotErrorInfo == nil {
		t.Fatal("status details missing ErrorInfo")
	}
	if got, want := gotErrorInfo.Reason, wantCode; got != want {
		t.Fatalf("error info reason = %q, want %q", got, want)
	}
	if got, want := gotErrorInfo.Domain, playReadinessErrorDomain; got != want {
		t.Fatalf("error info domain = %q, want %q", got, want)
	}

	if gotPrecondition == nil {
		t.Fatal("status details missing PreconditionFailure")
	}
	if got, want := len(gotPrecondition.Violations), 1; got != want {
		t.Fatalf("precondition violations len = %d, want %d", got, want)
	}
	violation := gotPrecondition.Violations[0]
	if got, want := violation.Type, wantCode; got != want {
		t.Fatalf("precondition type = %q, want %q", got, want)
	}
	if got, want := violation.Subject, playReadinessErrorSubject; got != want {
		t.Fatalf("precondition subject = %q, want %q", got, want)
	}
	if got, want := violation.Description, wantMessage; got != want {
		t.Fatalf("precondition description = %q, want %q", got, want)
	}
}

type unknownEvent struct{}

func (unknownEvent) EventType() event.Type { return "test.unknown" }

type panicTransportService struct{}

func (panicTransportService) CommitCommand(context.Context, caller.Caller, command.Envelope) (service.Result, error) {
	panic("unexpected Execute call")
}

func (panicTransportService) Inspect(context.Context, caller.Caller, string) (service.Inspection, error) {
	panic("unexpected Inspect call")
}

func (panicTransportService) ListCampaigns(context.Context, caller.Caller) ([]service.CampaignSummary, error) {
	panic("unexpected ListCampaigns call")
}

func (panicTransportService) GetPlayReadiness(context.Context, caller.Caller, string) (service.PlayReadiness, error) {
	panic("unexpected GetPlayReadiness call")
}

func (panicTransportService) PlanCommands(context.Context, caller.Caller, []command.Envelope) (service.CommandPlan, error) {
	panic("unexpected PlanCommands call")
}

func (panicTransportService) ExecutePlan(context.Context, caller.Caller, string) (service.ExecutedPlan, error) {
	panic("unexpected ExecutePlan call")
}

func (panicTransportService) ListEvents(context.Context, caller.Caller, string, uint64) ([]event.Record, error) {
	panic("unexpected ListEvents call")
}

func (panicTransportService) SubscribeEvents(context.Context, caller.Caller, string, uint64) (service.EventStream, error) {
	panic("unexpected SubscribeEvents call")
}

func (panicTransportService) ReadResource(context.Context, caller.Caller, string) (string, error) {
	panic("unexpected ReadResource call")
}

type fakeStream struct {
	gamev1pb.GameService_StreamCampaignEventsServer
	ctx     context.Context
	sent    []*gamev1pb.Event
	sendErr error
}

func (s *fakeStream) Context() context.Context {
	if s.ctx == nil {
		return inboundContext("subject-1")
	}
	return s.ctx
}

func (s *fakeStream) Send(event *gamev1pb.Event) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent = append(s.sent, event)
	return nil
}
