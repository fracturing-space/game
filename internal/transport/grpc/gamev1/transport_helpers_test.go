package gamev1

import (
	"context"
	"errors"
	"testing"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/readiness"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResultHelpers(t *testing.T) {
	t.Parallel()

	campaignUpdated, err := event.NewEnvelope(campaign.UpdatedEventSpec, "camp-1", campaign.Updated{Name: "Autumn Eclipse"})
	if err != nil {
		t.Fatalf("NewEnvelope(campaign updated) error = %v", err)
	}
	joined, err := event.NewEnvelope(participant.JoinedEventSpec, "camp-1", participant.Joined{ParticipantID: "part-2", Name: "Zoe", Access: participant.AccessMember})
	if err != nil {
		t.Fatalf("NewEnvelope(joined) error = %v", err)
	}
	bound, err := event.NewEnvelope(participant.BoundEventSpec, "camp-1", participant.Bound{ParticipantID: "part-2", SubjectID: "subject-2"})
	if err != nil {
		t.Fatalf("NewEnvelope(bound) error = %v", err)
	}
	updatedCharacter, err := event.NewEnvelope(character.UpdatedEventSpec, "camp-1", character.Updated{CharacterID: "char-1", ParticipantID: "part-1", Name: "Luna"})
	if err != nil {
		t.Fatalf("NewEnvelope(character updated) error = %v", err)
	}
	sceneCreated, err := event.NewEnvelope(scene.CreatedEventSpec, "camp-1", scene.Created{SceneID: "scene-1", SessionID: "sess-1", Name: "Opening", CharacterIDs: []string{"char-1"}})
	if err != nil {
		t.Fatalf("NewEnvelope(scene created) error = %v", err)
	}
	sceneEnded, err := event.NewEnvelope(scene.EndedEventSpec, "camp-1", scene.Ended{SceneID: "scene-1"})
	if err != nil {
		t.Fatalf("NewEnvelope(scene ended) error = %v", err)
	}
	sessionStarted, err := event.NewEnvelope(session.StartedEventSpec, "camp-1", session.Started{SessionID: "sess-1", Name: "Session 1"})
	if err != nil {
		t.Fatalf("NewEnvelope(session started) error = %v", err)
	}
	sessionEnded, err := event.NewEnvelope(session.EndedEventSpec, "camp-1", session.Ended{SessionID: "sess-1", Name: "Session 1"})
	if err != nil {
		t.Fatalf("NewEnvelope(session ended) error = %v", err)
	}
	unbound, err := event.NewEnvelope(participant.UnboundEventSpec, "camp-1", participant.Unbound{ParticipantID: "part-2"})
	if err != nil {
		t.Fatalf("NewEnvelope(unbound) error = %v", err)
	}
	left, err := event.NewEnvelope(participant.LeftEventSpec, "camp-1", participant.Left{ParticipantID: "part-2"})
	if err != nil {
		t.Fatalf("NewEnvelope(left) error = %v", err)
	}
	updatedParticipant, err := event.NewEnvelope(participant.UpdatedEventSpec, "camp-1", participant.Updated{ParticipantID: "part-2", Name: "Zoe Prime", Access: participant.AccessMember})
	if err != nil {
		t.Fatalf("NewEnvelope(updated participant) error = %v", err)
	}
	createdCharacter, err := event.NewEnvelope(character.CreatedEventSpec, "camp-1", character.Created{CharacterID: "char-1", ParticipantID: "part-1", Name: "Luna"})
	if err != nil {
		t.Fatalf("NewEnvelope(created character) error = %v", err)
	}
	deletedCharacter, err := event.NewEnvelope(character.DeletedEventSpec, "camp-1", character.Deleted{CharacterID: "char-1"})
	if err != nil {
		t.Fatalf("NewEnvelope(deleted character) error = %v", err)
	}
	sceneActivated, err := event.NewEnvelope(scene.ActivatedEventSpec, "camp-1", scene.Activated{SceneID: "scene-1"})
	if err != nil {
		t.Fatalf("NewEnvelope(scene activated) error = %v", err)
	}

	if got, err := resultCampaignID(service.Result{State: campaign.Snapshot{ID: "camp-1"}}); err != nil || got != "camp-1" {
		t.Fatalf("resultCampaignID(state) = (%q, %v), want camp-1", got, err)
	}
	if got, err := resultCampaignID(service.Result{Events: []event.Envelope{campaignUpdated}}); err != nil || got != "camp-1" {
		t.Fatalf("resultCampaignID(events) = (%q, %v), want camp-1", got, err)
	}
	if got, err := resultParticipantID(service.Result{StoredEvents: []event.Record{{Envelope: joined}}}); err != nil || got != "part-2" {
		t.Fatalf("resultParticipantID(stored) = (%q, %v), want part-2", got, err)
	}
	if got, err := resultParticipantEventID(service.Result{Events: []event.Envelope{bound}}); err != nil || got != "part-2" {
		t.Fatalf("resultParticipantEventID(events) = (%q, %v), want part-2", got, err)
	}
	if got, err := resultParticipantEventID(service.Result{StoredEvents: []event.Record{{Envelope: unbound}}}); err != nil || got != "part-2" {
		t.Fatalf("resultParticipantEventID(unbound) = (%q, %v), want part-2", got, err)
	}
	if got, err := resultParticipantEventID(service.Result{StoredEvents: []event.Record{{Envelope: left}}}); err != nil || got != "part-2" {
		t.Fatalf("resultParticipantEventID(left) = (%q, %v), want part-2", got, err)
	}
	if got, err := resultParticipantEventID(service.Result{StoredEvents: []event.Record{{Envelope: updatedParticipant}}}); err != nil || got != "part-2" {
		t.Fatalf("resultParticipantEventID(updated) = (%q, %v), want part-2", got, err)
	}
	if got, err := resultCharacterID(service.Result{Events: []event.Envelope{createdCharacter}}); err != nil || got != "char-1" {
		t.Fatalf("resultCharacterID(events) = (%q, %v), want char-1", got, err)
	}
	if got, err := resultCharacterEventID(service.Result{Events: []event.Envelope{updatedCharacter}}); err != nil || got != "char-1" {
		t.Fatalf("resultCharacterEventID(events) = (%q, %v), want char-1", got, err)
	}
	if got, err := resultCharacterEventID(service.Result{StoredEvents: []event.Record{{Envelope: deletedCharacter}}}); err != nil || got != "char-1" {
		t.Fatalf("resultCharacterEventID(stored) = (%q, %v), want char-1", got, err)
	}
	if got, err := resultSceneID(service.Result{Events: []event.Envelope{sceneCreated}}); err != nil || got != "scene-1" {
		t.Fatalf("resultSceneID(events) = (%q, %v), want scene-1", got, err)
	}
	if got, err := resultSceneEventID(service.Result{Events: []event.Envelope{sceneActivated}}); err != nil || got != "scene-1" {
		t.Fatalf("resultSceneEventID(events) = (%q, %v), want scene-1", got, err)
	}
	if got, err := resultSceneEventID(service.Result{StoredEvents: []event.Record{{Envelope: sceneEnded}}}); err != nil || got != "scene-1" {
		t.Fatalf("resultSceneEventID(stored) = (%q, %v), want scene-1", got, err)
	}
	if got, err := resultSession(service.Result{Events: []event.Envelope{sessionStarted}}); err != nil || got.GetId() != "sess-1" {
		t.Fatalf("resultSession(started) = (%v, %v), want sess-1", got, err)
	}
	if got, err := resultSession(service.Result{StoredEvents: []event.Record{{Envelope: sessionEnded}}}); err != nil || got.GetId() != "sess-1" {
		t.Fatalf("resultSession(ended) = (%v, %v), want sess-1", got, err)
	}
	if got, err := resultScene(service.Result{State: campaign.Snapshot{Scenes: []scene.Record{{ID: "scene-1", Name: "Opening"}}}}, "scene-1"); err != nil || got.GetId() != "scene-1" {
		t.Fatalf("resultScene() = (%v, %v), want scene-1", got, err)
	}
	if _, err := resultScene(service.Result{}, " scene-1 "); err == nil {
		t.Fatal("resultScene(padded scene id) error = nil, want failure")
	}

	if _, err := resultCampaignID(service.Result{}); err == nil {
		t.Fatal("resultCampaignID(empty) error = nil, want failure")
	}
	if _, err := resultParticipantID(service.Result{}); err == nil {
		t.Fatal("resultParticipantID(empty) error = nil, want failure")
	}
	if _, err := resultParticipantEventID(service.Result{}); err == nil {
		t.Fatal("resultParticipantEventID(empty) error = nil, want failure")
	}
	if _, err := resultCharacterID(service.Result{}); err == nil {
		t.Fatal("resultCharacterID(empty) error = nil, want failure")
	}
	if _, err := resultCharacterEventID(service.Result{}); err == nil {
		t.Fatal("resultCharacterEventID(empty) error = nil, want failure")
	}
	if _, err := resultSceneID(service.Result{}); err == nil {
		t.Fatal("resultSceneID(empty) error = nil, want failure")
	}
	if _, err := resultSceneEventID(service.Result{}); err == nil {
		t.Fatal("resultSceneEventID(empty) error = nil, want failure")
	}
	if _, err := resultScene(service.Result{}, "scene-1"); err == nil {
		t.Fatal("resultScene(empty) error = nil, want failure")
	}
	if _, err := resultSession(service.Result{}); err == nil {
		t.Fatal("resultSession(empty) error = nil, want failure")
	}
}

func TestDomainAndReadinessTransportHelpers(t *testing.T) {
	t.Parallel()

	if got := invalidArgument(nil); got != nil {
		t.Fatalf("invalidArgument(nil) = %v, want nil", got)
	}
	if got := internalStatus(nil); got != nil {
		t.Fatalf("internalStatus(nil) = %v, want nil", got)
	}

	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "canceled", err: context.Canceled, want: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
		{name: "denied", err: &authz.DeniedError{Capability: authz.CapabilityReadCampaign, Reason: "nope"}, want: codes.PermissionDenied},
		{name: "not found", err: errs.NotFoundf("missing"), want: codes.NotFound},
		{name: "already exists", err: errs.AlreadyExistsf("exists"), want: codes.AlreadyExists},
		{name: "conflict", err: errs.Conflictf("stale"), want: codes.Aborted},
		{name: "failed precondition", err: errs.FailedPreconditionf("bad"), want: codes.FailedPrecondition},
		{name: "invalid", err: errs.InvalidArgumentf("bad"), want: codes.InvalidArgument},
		{name: "fallback", err: errors.New("boom"), want: codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := status.Code(mapDomainError(tc.err)); got != tc.want {
				t.Fatalf("mapDomainError(%s) code = %v, want %v", tc.name, got, tc.want)
			}
		})
	}

	assertPlayReadinessStatusDetails(t, mapPlayReadinessError(&readiness.Rejection{
		Code:    readiness.RejectionCodePlayReadinessPlayerRequired,
		Message: "at least one bound player is required before entering PLAY",
	}), readiness.RejectionCodePlayReadinessPlayerRequired, "at least one bound player is required before entering PLAY")

	if got := status.Code(mapPlayReadinessError(errors.New("boom"))); got != codes.FailedPrecondition {
		t.Fatalf("mapPlayReadinessError(non rejection) code = %v, want %v", got, codes.FailedPrecondition)
	}
	if _, err := protoPlayReadinessAction(readiness.Action{ResolutionKind: readiness.ResolutionKind("broken")}); err == nil {
		t.Fatal("protoPlayReadinessAction(invalid kind) error = nil, want failure")
	}
	if _, err := protoPlayReadiness(service.PlayReadiness{Blockers: []readiness.Blocker{{
		Code:    "X",
		Message: "broken",
		Action:  readiness.Action{ResolutionKind: readiness.ResolutionKind("broken")},
	}}}); err == nil {
		t.Fatal("protoPlayReadiness(invalid kind) error = nil, want failure")
	}
}
