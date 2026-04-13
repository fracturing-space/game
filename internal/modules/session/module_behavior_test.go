package session

import (
	"testing"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestDecideEndWithSceneAndAssignmentEdgeCases(t *testing.T) {
	t.Parallel()

	state := readyCampaignState()
	state.PlayState = campaign.PlayStateActive
	state.Sessions["sess-1"] = session.Record{
		ID:                   "sess-1",
		Name:                 "Night Watch",
		Status:               session.StatusActive,
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}
	state.ActiveSessionID = "sess-1"
	state.ActiveSceneID = "scene-1"
	state.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening", Active: true}

	events, err := decideEnd(state, command.Envelope{CampaignID: "camp-1", Message: session.End{}})
	if err != nil {
		t.Fatalf("decideEnd() error = %v", err)
	}
	if got, want := len(events), 3; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
	if _, err := event.MessageAs[scene.Ended](events[0]); err != nil {
		t.Fatalf("first event should be scene.Ended: %v", err)
	}
	if _, err := event.MessageAs[session.Ended](events[1]); err != nil {
		t.Fatalf("second event should be session.Ended: %v", err)
	}
	if _, err := event.MessageAs[campaign.PlayEnded](events[2]); err != nil {
		t.Fatalf("third event should be campaign.PlayEnded: %v", err)
	}

	state.ActiveSceneID = "missing"
	events, err = decideEnd(state, command.Envelope{CampaignID: "camp-1", Message: session.End{}})
	if err != nil {
		t.Fatalf("decideEnd(missing scene record) error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("events len without scene record = %d, want %d", got, want)
	}
	state.ActiveSceneID = "scene-1"
	state.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening", Active: true}
	if _, err := decideEnd(state, command.Envelope{Message: session.End{}}); err == nil {
		t.Fatal("decideEnd(missing campaign id with active scene) error = nil, want failure")
	}
}

func TestEffectiveAssignmentsEdgeCases(t *testing.T) {
	t.Parallel()

	state := readyCampaignState()
	state.Characters["inactive-char"] = character.Record{ID: "inactive-char", ParticipantID: "part-1", Name: "Ghost"}

	assignments, err := effectiveAssignments(state, nil)
	if err != nil {
		t.Fatalf("effectiveAssignments() error = %v", err)
	}
	if got, want := len(assignments), 1; got != want {
		t.Fatalf("assignments len = %d, want %d", got, want)
	}

	state.Characters["char-blank-owner"] = character.Record{ID: "char-blank-owner", Name: "Ghost", Active: true}
	if _, err := effectiveAssignments(state, nil); err == nil {
		t.Fatal("effectiveAssignments(blank owner) error = nil, want failure")
	}
	delete(state.Characters, "char-blank-owner")

	state.Characters["char-1"] = character.Record{
		ID:            "char-1",
		ParticipantID: "inactive-participant",
		Name:          "Luna", Active: true,
	}
	state.Participants["inactive-participant"] = participant.Record{ID: "inactive-participant"}
	if _, err := effectiveAssignments(state, nil); err == nil {
		t.Fatal("effectiveAssignments(inactive owner) error = nil, want failure")
	}
	state = readyCampaignState()
	if _, err := effectiveAssignments(state, []session.CharacterControllerAssignment{{CharacterID: "char-1"}}); err == nil {
		t.Fatal("effectiveAssignments(blank override participant) error = nil, want failure")
	}
}
