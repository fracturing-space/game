package campaign

import (
	"testing"

	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestSnapshotSceneHelpers(t *testing.T) {
	t.Parallel()

	state := NewState()
	state.ActiveSceneID = "scene-2"
	state.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening", CharacterIDs: []string{"char-1"}}
	state.Scenes["scene-2"] = scene.Record{ID: "scene-2", SessionID: "sess-2", Name: "Harbor", CharacterIDs: []string{"char-2"}}
	state.Scenes["scene-3"] = scene.Record{ID: "scene-3", SessionID: "sess-2", Name: "Market", CharacterIDs: []string{"char-3"}}

	snapshot := SnapshotOf(state)

	active := snapshot.ActiveScene()
	if active == nil || active.ID != "scene-2" {
		t.Fatalf("ActiveScene() = %+v, want scene-2", active)
	}
	active.Name = "mutated"
	active.CharacterIDs[0] = "changed"

	activeAgain := snapshot.ActiveScene()
	if activeAgain == nil || activeAgain.Name != "Harbor" {
		t.Fatalf("ActiveScene() clone = %+v, want original name", activeAgain)
	}
	if got, want := activeAgain.CharacterIDs[0], "char-2"; got != want {
		t.Fatalf("ActiveScene() clone character = %q, want %q", got, want)
	}

	scenes := snapshot.ScenesForSession("sess-2")
	if got, want := len(scenes), 2; got != want {
		t.Fatalf("ScenesForSession() len = %d, want %d", got, want)
	}
	scenes[0].Name = "mutated"
	scenes[0].CharacterIDs[0] = "changed"

	scenesAgain := snapshot.ScenesForSession("sess-2")
	if got, want := len(scenesAgain), 2; got != want {
		t.Fatalf("ScenesForSession(second) len = %d, want %d", got, want)
	}
	if scenesAgain[0].Name == "mutated" || scenesAgain[0].CharacterIDs[0] == "changed" {
		t.Fatal("ScenesForSession() should return clone-safe records")
	}

	empty := Snapshot{}
	if empty.ActiveScene() != nil {
		t.Fatal("ActiveScene(empty) = non-nil, want nil")
	}
	if got := empty.ScenesForSession("sess-1"); len(got) != 0 {
		t.Fatalf("ScenesForSession(empty) len = %d, want 0", len(got))
	}
}

func TestValidatePlayStateTransitionAllowedBranches(t *testing.T) {
	t.Parallel()

	allowed := [][2]PlayState{
		{PlayStateSetup, PlayStateActive},
		{PlayStateActive, PlayStatePaused},
		{PlayStatePaused, PlayStateActive},
		{PlayStateActive, PlayStateSetup},
		{PlayStatePaused, PlayStateSetup},
	}
	for _, transition := range allowed {
		if err := ValidatePlayStateTransition(transition[0], transition[1]); err != nil {
			t.Fatalf("ValidatePlayStateTransition(%s -> %s) error = %v", transition[0], transition[1], err)
		}
	}
}

func TestSnapshotSessionHelpers(t *testing.T) {
	t.Parallel()

	state := NewState()
	state.ActiveSessionID = "sess-2"
	state.Sessions["sess-1"] = session.Record{
		ID:     "sess-1",
		Name:   "Downtime",
		Status: session.StatusEnded,
		CharacterControllers: []session.CharacterControllerAssignment{
			{CharacterID: "char-1", ParticipantID: "part-1"},
		},
	}
	state.Sessions["sess-2"] = session.Record{
		ID:     "sess-2",
		Name:   "Harbor",
		Status: session.StatusActive,
		CharacterControllers: []session.CharacterControllerAssignment{
			{CharacterID: "char-2", ParticipantID: "part-2"},
		},
	}

	activeState := state.ActiveSession()
	if activeState == nil || activeState.ID != "sess-2" {
		t.Fatalf("State.ActiveSession() = %+v, want sess-2", activeState)
	}
	activeState.Name = "mutated"
	activeState.CharacterControllers[0].ParticipantID = "changed"

	activeStateAgain := state.ActiveSession()
	if activeStateAgain == nil || activeStateAgain.Name != "Harbor" {
		t.Fatalf("State.ActiveSession() clone = %+v, want original name", activeStateAgain)
	}
	if got, want := activeStateAgain.CharacterControllers[0].ParticipantID, "part-2"; got != want {
		t.Fatalf("State.ActiveSession() clone participant = %q, want %q", got, want)
	}

	snapshot := SnapshotOf(state)
	if got, want := len(snapshot.Sessions), 2; got != want {
		t.Fatalf("SnapshotOf().Sessions len = %d, want %d", got, want)
	}

	activeSnapshot := snapshot.ActiveSession()
	if activeSnapshot == nil || activeSnapshot.ID != "sess-2" {
		t.Fatalf("Snapshot.ActiveSession() = %+v, want sess-2", activeSnapshot)
	}
	activeSnapshot.Name = "mutated"
	activeSnapshot.CharacterControllers[0].ParticipantID = "changed"

	activeSnapshotAgain := snapshot.ActiveSession()
	if activeSnapshotAgain == nil || activeSnapshotAgain.Name != "Harbor" {
		t.Fatalf("Snapshot.ActiveSession() clone = %+v, want original name", activeSnapshotAgain)
	}
	if got, want := activeSnapshotAgain.CharacterControllers[0].ParticipantID, "part-2"; got != want {
		t.Fatalf("Snapshot.ActiveSession() clone participant = %q, want %q", got, want)
	}

	found := snapshot.Session("sess-1")
	if found == nil || found.ID != "sess-1" {
		t.Fatalf("Snapshot.Session(sess-1) = %+v, want sess-1", found)
	}
	found.Name = "mutated"

	foundAgain := snapshot.Session("sess-1")
	if foundAgain == nil || foundAgain.Name != "Downtime" {
		t.Fatalf("Snapshot.Session(sess-1) clone = %+v, want original name", foundAgain)
	}

	emptyState := State{}
	if emptyState.ActiveSession() != nil {
		t.Fatal("State.ActiveSession(empty) = non-nil, want nil")
	}

	emptySnapshot := Snapshot{}
	if emptySnapshot.ActiveSession() != nil {
		t.Fatal("Snapshot.ActiveSession(empty) = non-nil, want nil")
	}
	if emptySnapshot.Session("") != nil {
		t.Fatal("Snapshot.Session(empty) = non-nil, want nil")
	}
	if emptySnapshot.Session("missing") != nil {
		t.Fatal("Snapshot.Session(missing) = non-nil, want nil")
	}
}
