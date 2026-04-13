package campaign

import (
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
)

func TestStateHelpers(t *testing.T) {
	t.Parallel()

	state := NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "Autumn Twilight"
	state.PlayState = PlayStateSetup
	state.Characters["char-2"] = character.Record{ID: "char-2", ParticipantID: "part-2", Name: "zoe", Active: true}
	state.Characters["char-1"] = character.Record{ID: "char-1", ParticipantID: "part-1", Name: "louis", Active: true}
	state.Participants["part-2"] = participant.Record{ID: "part-2", Name: "zoe", Access: participant.AccessMember, SubjectID: "subject-2", Active: true}
	state.Participants["part-1"] = participant.Record{ID: "part-1", Name: "louis", Access: participant.AccessOwner, SubjectID: "subject-1", Active: true}
	state.AIAgentID = "agent-1"
	state.Scenes["scene-1"] = scene.Record{ID: "scene-1", Name: "Opening Scene", CharacterIDs: []string{"char-1"}}

	clone := state.Clone()
	clone.Characters["char-1"] = character.Record{ID: "char-1", ParticipantID: "part-9", Name: "changed", Active: true}
	if got, want := state.Characters["char-1"].Name, "louis"; got != want {
		t.Fatalf("Clone() changed original character to %q, want %q", got, want)
	}

	snapshot := SnapshotOf(state)
	if got, want := len(snapshot.Characters), 2; got != want {
		t.Fatalf("characters len = %d, want %d", got, want)
	}
	if got, want := len(snapshot.Participants), 2; got != want {
		t.Fatalf("participants len = %d, want %d", got, want)
	}
}

func TestPlayStateValidation(t *testing.T) {
	t.Parallel()

	if err := ValidatePlayStateTransition(PlayStateSetup, PlayStateActive); err != nil {
		t.Fatalf("ValidatePlayStateTransition(setup->active) error = %v", err)
	}
	if err := ValidatePlayStateTransition(PlayStateSetup, PlayStateSetup); err == nil {
		t.Fatal("ValidatePlayStateTransition(setup->setup) error = nil, want failure")
	}
	if err := ValidatePlayStateTransition(PlayState("BROKEN"), PlayStateActive); err == nil {
		t.Fatal("ValidatePlayStateTransition(invalid from) error = nil, want failure")
	}
	if err := ValidatePlayStateTransition(PlayStateSetup, PlayState("BROKEN")); err == nil {
		t.Fatal("ValidatePlayStateTransition(invalid to) error = nil, want failure")
	}
	if !PlayStateSetup.Valid() || !PlayStateActive.Valid() || !PlayStatePaused.Valid() {
		t.Fatal("expected built-in play states to validate")
	}
	if PlayState("").Valid() {
		t.Fatal("empty play state should be invalid")
	}
}

func TestValidationAndHelpers(t *testing.T) {
	t.Parallel()

	if err := ValidateCreate(Create{Name: "Autumn Twilight", OwnerName: "louis"}); err != nil {
		t.Fatalf("ValidateCreate() error = %v", err)
	}
	if err := ValidateUpdate(Update{Name: "Autumn Dusk"}); err != nil {
		t.Fatalf("ValidateUpdate(valid) error = %v", err)
	}
	if err := ValidateCreated(Created{Name: "Autumn Twilight"}); err != nil {
		t.Fatalf("ValidateCreated(valid) error = %v", err)
	}
	if err := ValidateUpdated(Updated{Name: "Autumn Dusk"}); err != nil {
		t.Fatalf("ValidateUpdated(valid) error = %v", err)
	}
	if err := ValidateAIBind(AIBind{AIAgentID: "agent-1"}); err != nil {
		t.Fatalf("ValidateAIBind() error = %v", err)
	}
	if err := ValidateAIBound(AIBound{AIAgentID: "agent-1"}); err != nil {
		t.Fatalf("ValidateAIBound(valid) error = %v", err)
	}
	if err := ValidatePlayBegan(PlayBegan{SessionID: "sess-1", SceneID: "scene-1"}); err != nil {
		t.Fatalf("ValidatePlayBegan(valid) error = %v", err)
	}
	if err := ValidatePlayPaused(PlayPaused{SessionID: "sess-1", SceneID: "scene-1", Reason: "break"}); err != nil {
		t.Fatalf("ValidatePlayPaused(valid) error = %v", err)
	}
	if err := ValidatePlayResumed(PlayResumed{SessionID: "sess-1", SceneID: "scene-1", Reason: "resume"}); err != nil {
		t.Fatalf("ValidatePlayResumed(valid) error = %v", err)
	}
	if !HasBoundAIAgent(State{AIAgentID: "agent-1"}) {
		t.Fatal("HasBoundAIAgent() = false, want true")
	}
	if HasBoundAIAgent(State{}) {
		t.Fatal("HasBoundAIAgent(empty) = true, want false")
	}
	if err := ValidateCreate(Create{}); err == nil {
		t.Fatal("ValidateCreate(empty) error = nil, want failure")
	}
	if err := ValidateUpdate(Update{}); err == nil {
		t.Fatal("ValidateUpdate(empty) error = nil, want failure")
	}
	if err := ValidateCreated(Created{}); err == nil {
		t.Fatal("ValidateCreated(empty) error = nil, want failure")
	}
	if err := ValidateUpdated(Updated{}); err == nil {
		t.Fatal("ValidateUpdated(empty) error = nil, want failure")
	}
	if err := ValidateAIBound(AIBound{}); err == nil {
		t.Fatal("ValidateAIBound(empty) error = nil, want failure")
	}
	if err := ValidatePlayBegan(PlayBegan{}); err == nil {
		t.Fatal("ValidatePlayBegan(empty) error = nil, want failure")
	}
	if err := ValidatePlayPaused(PlayPaused{SessionID: "sess-1"}); err == nil {
		t.Fatal("ValidatePlayPaused(missing scene) error = nil, want failure")
	}
	if err := ValidatePlayResumed(PlayResumed{SessionID: "sess-1"}); err == nil {
		t.Fatal("ValidatePlayResumed(missing scene) error = nil, want failure")
	}
}

func TestCampaignNormalizersAndAuthorityHelpers(t *testing.T) {
	t.Parallel()

	if got := normalizeCreate(Create{Name: " Autumn Twilight ", OwnerName: " louis "}); got.Name != "Autumn Twilight" || got.OwnerName != "louis" {
		t.Fatalf("normalizeCreate() = %+v, want trimmed fields", got)
	}
	if got := normalizeUpdate(Update{Name: " Autumn Dusk "}); got.Name != "Autumn Dusk" {
		t.Fatalf("normalizeUpdate() = %+v, want trimmed name", got)
	}
	if got := normalizeAIBind(AIBind{AIAgentID: "agent-1"}); got.AIAgentID != "agent-1" {
		t.Fatalf("normalizeAIBind() = %+v, want passthrough", got)
	}
	if got := normalizePlayPause(PlayPause{Reason: " break "}); got.Reason != "break" {
		t.Fatalf("normalizePlayPause() = %+v, want trimmed reason", got)
	}
	if got := normalizePlayResume(PlayResume{Reason: " resume "}); got.Reason != "resume" {
		t.Fatalf("normalizePlayResume() = %+v, want trimmed reason", got)
	}
	if got := normalizeCreated(Created{Name: " Autumn Twilight "}); got.Name != "Autumn Twilight" {
		t.Fatalf("normalizeCreated() = %+v, want trimmed name", got)
	}
	if got := normalizeUpdated(Updated{Name: " Autumn Dusk "}); got.Name != "Autumn Dusk" {
		t.Fatalf("normalizeUpdated() = %+v, want trimmed name", got)
	}
	if got := normalizeAIBound(AIBound{AIAgentID: "agent-1"}); got.AIAgentID != "agent-1" {
		t.Fatalf("normalizeAIBound() = %+v, want passthrough", got)
	}
	if got := normalizePlayBegan(PlayBegan{SessionID: "sess-1", SceneID: "scene-1"}); got.SessionID != "sess-1" || got.SceneID != "scene-1" {
		t.Fatalf("normalizePlayBegan() = %+v, want passthrough", got)
	}
	if got := normalizePlayPaused(PlayPaused{Reason: " break "}); got.Reason != "break" {
		t.Fatalf("normalizePlayPaused() = %+v, want trimmed reason", got)
	}
	if got := normalizePlayResumed(PlayResumed{Reason: " resume "}); got.Reason != "resume" {
		t.Fatalf("normalizePlayResumed() = %+v, want trimmed reason", got)
	}
	if got := normalizePlayEnded(PlayEnded{SessionID: "sess-1", SceneID: "scene-1"}); got.SessionID != "sess-1" || got.SceneID != "scene-1" {
		t.Fatalf("normalizePlayEnded() = %+v, want passthrough", got)
	}

	state := NewState()
	state.Participants["owner-1"] = participant.Record{
		ID:        "owner-1",
		Access:    participant.AccessOwner,
		SubjectID: "subject-1",
		Active:    true,
	}
	if !HasBoundSubject(state, "subject-1") {
		t.Fatal("HasBoundSubject(active) = false, want true")
	}
	if HasBoundSubject(state, "") {
		t.Fatal("HasBoundSubject(empty) = true, want false")
	}
	if _, ok := BoundParticipant(state, ""); ok {
		t.Fatal("BoundParticipant(empty) = true, want false")
	}
	if record, ok := BoundParticipant(state, "subject-1"); !ok || record.ID != "owner-1" {
		t.Fatalf("BoundParticipant() = (%+v, %t), want owner-1", record, ok)
	}
	if _, ok := CallerParticipant(state, caller.Caller{}); ok {
		t.Fatal("CallerParticipant(no subject) = true, want false")
	}
}
