package campaign

import (
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
)

func TestCampaignUpdateAndSnapshotFiltering(t *testing.T) {
	t.Parallel()

	state := NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "Autumn Twilight"
	state.PlayState = PlayStateSetup
	state.ActiveSceneID = "scene-2"
	state.Characters["char-1"] = character.Record{ID: "char-1", ParticipantID: "part-1", Name: "Luna", Active: true}
	state.Characters["char-2"] = character.Record{ID: "char-2", ParticipantID: "part-2", Name: "Ghost"}
	state.Participants["part-1"] = participant.Record{ID: "part-1", Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1", Active: true}
	state.Participants["part-2"] = participant.Record{ID: "part-2", Name: "Ghost", Access: participant.AccessMember}
	state.Scenes["scene-2"] = scene.Record{ID: "scene-2", Name: "Second"}
	state.Scenes["scene-1"] = scene.Record{ID: "scene-1", Name: "First"}

	snapshot := SnapshotOf(state)
	if got, want := len(snapshot.Characters), 1; got != want {
		t.Fatalf("characters len = %d, want %d", got, want)
	}
	if got, want := len(snapshot.Participants), 1; got != want {
		t.Fatalf("participants len = %d, want %d", got, want)
	}
}

func TestCampaignParticipantHelpersIgnoreInactiveRecords(t *testing.T) {
	t.Parallel()

	state := NewState()
	state.Participants["inactive-owner"] = participant.Record{
		ID:        "inactive-owner",
		Access:    participant.AccessOwner,
		SubjectID: "subject-1",
	}
	if HasBoundSubject(state, "subject-1") {
		t.Fatal("HasBoundSubject(inactive) = true, want false")
	}
	if _, ok := BoundParticipant(state, "subject-1"); ok {
		t.Fatal("BoundParticipant(inactive) = true, want false")
	}
}

func TestCallerHelpers(t *testing.T) {
	t.Parallel()

	state := NewState()
	state.AIAgentID = "agent-1"
	state.Participants["owner-1"] = participant.Record{
		ID:        "owner-1",
		Name:      "Owner",
		Access:    participant.AccessOwner,
		SubjectID: "subject-owner",
		Active:    true,
	}

	if record, ok := CallerParticipant(state, caller.MustNewSubject("subject-owner")); !ok || record.ID != "owner-1" {
		t.Fatalf("CallerParticipant() = (%+v, %t), want owner-1", record, ok)
	}
	if !CallerMatchesBoundAIAgent(state, caller.MustNewAIAgent("agent-1")) {
		t.Fatal("CallerMatchesBoundAIAgent(match) = false, want true")
	}
	if CallerMatchesBoundAIAgent(state, caller.MustNewAIAgent("agent-2")) {
		t.Fatal("CallerMatchesBoundAIAgent(mismatch) = true, want false")
	}
}
