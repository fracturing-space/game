package gamev1

import (
	"testing"
	"time"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
)

func TestProtoEventCoversSupportedEventPayloads(t *testing.T) {
	t.Parallel()

	recordedAt := time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)
	envelopes := []event.Envelope{
		mustTransportEnvelope(t, "camp-1", campaign.Created{Name: "Autumn Twilight"}),
		mustTransportEnvelope(t, "camp-1", campaign.Updated{Name: "Autumn Eclipse"}),
		mustTransportEnvelope(t, "camp-1", campaign.PlayBegan{SessionID: "sess-1", SceneID: "scene-1"}),
		mustTransportEnvelope(t, "camp-1", campaign.PlayPaused{SessionID: "sess-1", SceneID: "scene-1", Reason: "rules"}),
		mustTransportEnvelope(t, "camp-1", campaign.PlayResumed{SessionID: "sess-1", SceneID: "scene-1", Reason: "resume"}),
		mustTransportEnvelope(t, "camp-1", campaign.PlayEnded{SessionID: "sess-1", SceneID: "scene-1"}),
		mustTransportEnvelope(t, "camp-1", campaign.AIBound{AIAgentID: "agent-7"}),
		mustTransportEnvelope(t, "camp-1", campaign.AIUnbound{}),
		mustTransportEnvelope(t, "camp-1", character.Created{CharacterID: "char-1", ParticipantID: "part-1", Name: "Luna"}),
		mustTransportEnvelope(t, "camp-1", character.Updated{CharacterID: "char-1", ParticipantID: "part-1", Name: "Luna Prime"}),
		mustTransportEnvelope(t, "camp-1", character.Deleted{CharacterID: "char-1"}),
		mustTransportEnvelope(t, "camp-1", session.Started{SessionID: "sess-1", Name: "Session 1", CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}}}),
		mustTransportEnvelope(t, "camp-1", session.Ended{SessionID: "sess-1", Name: "Session 1", CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}}}),
		mustTransportEnvelope(t, "camp-1", participant.Joined{ParticipantID: "part-2", Name: "Zoe", Access: participant.AccessMember}),
		mustTransportEnvelope(t, "camp-1", participant.Updated{ParticipantID: "part-2", Name: "Zoe Prime", Access: participant.AccessMember}),
		mustTransportEnvelope(t, "camp-1", participant.Bound{ParticipantID: "part-2", SubjectID: "subject-2"}),
		mustTransportEnvelope(t, "camp-1", participant.Unbound{ParticipantID: "part-2"}),
		mustTransportEnvelope(t, "camp-1", participant.Left{ParticipantID: "part-2"}),
		mustTransportEnvelope(t, "camp-1", scene.Created{SceneID: "scene-1", SessionID: "sess-1", Name: "Opening", CharacterIDs: []string{"char-1"}}),
		mustTransportEnvelope(t, "camp-1", scene.Activated{SceneID: "scene-1"}),
		mustTransportEnvelope(t, "camp-1", scene.Ended{SceneID: "scene-1"}),
		mustTransportEnvelope(t, "camp-1", scene.CastReplaced{SceneID: "scene-1", CharacterIDs: []string{"char-1", "char-2"}}),
	}

	for _, envelope := range envelopes {
		t.Run(string(envelope.Type()), func(t *testing.T) {
			plannedProto, err := protoPlannedEvent(envelope)
			if err != nil {
				t.Fatalf("protoPlannedEvent(%s) error = %v", envelope.Type(), err)
			}
			if plannedProto.GetPayload() == nil {
				t.Fatalf("protoPlannedEvent(%s) payload = nil, want payload", envelope.Type())
			}

			storedProto, err := protoStoredEvent(event.Record{
				Seq:        7,
				CommitSeq:  3,
				RecordedAt: recordedAt,
				Envelope:   envelope,
			})
			if err != nil {
				t.Fatalf("protoStoredEvent(%s) error = %v", envelope.Type(), err)
			}
			if got, want := storedProto.GetSeq(), uint64(7); got != want {
				t.Fatalf("stored seq = %d, want %d", got, want)
			}
			if storedProto.GetPayload() == nil {
				t.Fatalf("protoStoredEvent(%s) payload = nil, want payload", envelope.Type())
			}
			if storedProto.GetRecordedAt() == nil {
				t.Fatalf("protoStoredEvent(%s) recorded at = nil, want timestamp", envelope.Type())
			}
		})
	}
}

func TestEventPayloadEncodersCoverDefaultCoreModuleEvents(t *testing.T) {
	t.Parallel()

	for _, module := range service.DefaultCoreModules() {
		for _, spec := range module.Events() {
			if _, ok := eventPayloadEncoders[spec.Definition().Type]; !ok {
				t.Fatalf("event payload encoder missing for %s", spec.Definition().Type)
			}
		}
	}
}

func TestProtoEventAndHelpersRejectInvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := protoEvent(0, 0, event.Type("core.unknown"), "camp-1", campaign.Created{Name: "Autumn Twilight"}, nil); err == nil {
		t.Fatal("protoEvent(unknown type) error = nil, want failure")
	}
	if _, err := protoEvent(0, 0, campaign.EventTypeCreated, "camp-1", campaign.Updated{Name: "Autumn Twilight"}, nil); err == nil {
		t.Fatal("protoEvent(mismatched payload) error = nil, want failure")
	}
	if _, err := protoAccess(participant.Access("broken")); err == nil {
		t.Fatal("protoAccess(invalid) error = nil, want failure")
	}
	if _, err := protoCampaignPlayState(campaign.PlayState("broken")); err == nil {
		t.Fatal("protoCampaignPlayState(invalid) error = nil, want failure")
	}
	if _, err := protoSessionStatus(session.Status("broken")); err == nil {
		t.Fatal("protoSessionStatus(invalid) error = nil, want failure")
	}
	if _, err := protoSession("sess-1", "Session 1", session.Status("broken"), nil); err == nil {
		t.Fatal("protoSession(invalid status) error = nil, want failure")
	}
	if _, err := protoActiveSession(&session.Record{ID: "sess-1", Name: "Session 1", Status: session.Status("broken")}); err == nil {
		t.Fatal("protoActiveSession(invalid status) error = nil, want failure")
	}
	if _, err := domainAccess(gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_UNSPECIFIED); err == nil {
		t.Fatal("domainAccess(invalid) error = nil, want failure")
	}
}
