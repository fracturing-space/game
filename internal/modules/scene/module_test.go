package scene

import (
	"errors"
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestModuleMetadataAndAdmission(t *testing.T) {
	t.Parallel()

	module := New()
	if got, want := module.Name(), "core.scene"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if got, want := len(module.Commands()), 4; got != want {
		t.Fatalf("Commands() len = %d, want %d", got, want)
	}
	if got, want := len(module.Events()), 4; got != want {
		t.Fatalf("Events() len = %d, want %d", got, want)
	}

	state := readySceneCampaignState()
	for _, registration := range module.Commands() {
		if got, want := len(registration.Admission.AllowedPlayStates), 1; got != want {
			t.Fatalf("AllowedPlayStates len = %d, want %d", got, want)
		}
		if got, want := registration.Admission.AllowedPlayStates[0], campaign.PlayStateActive; got != want {
			t.Fatalf("AllowedPlayStates[0] = %q, want %q", got, want)
		}
		if err := registration.Admission.Authorize(caller.MustNewAIAgent("agent-1"), state); err != nil {
			t.Fatalf("Admission.Authorize(gm authority) error = %v", err)
		}
		if err := registration.Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err == nil {
			t.Fatal("Admission.Authorize(owner proxy) error = nil, want failure")
		}
	}
}

func TestModuleDecideAndFold(t *testing.T) {
	t.Parallel()

	module := New()
	state := readySceneCampaignState()

	if _, err := module.Decide(campaign.NewState(), caller.MustNewAIAgent("agent-1"), command.Envelope{Message: testSceneCommand{}}, staticSceneIDs()); err == nil {
		t.Fatal("Decide(unknown) error = nil, want failure")
	}
	if _, err := decideCreate(campaign.NewState(), command.Envelope{CampaignID: "camp-1", Message: scene.Create{Name: "Opening"}}, staticSceneIDs("scene-1")); err == nil {
		t.Fatal("decideCreate(missing campaign) error = nil, want failure")
	}
	if _, err := decideCreate(campaign.State{Exists: true, CampaignID: "camp-1"}, command.Envelope{CampaignID: "camp-1", Message: scene.Create{Name: "Opening"}}, staticSceneIDs("scene-1")); err == nil {
		t.Fatal("decideCreate(missing session) error = nil, want failure")
	}
	if _, err := decideCreate(state, command.Envelope{CampaignID: "camp-1", Message: testSceneCommand{}}, staticSceneIDs("scene-1")); err == nil {
		t.Fatal("decideCreate(bad message) error = nil, want failure")
	}
	if _, err := decideCreate(state, command.Envelope{CampaignID: "camp-1", Message: scene.Create{Name: "Opening", CharacterIDs: []string{"missing"}}}, staticSceneIDs("scene-1")); err == nil {
		t.Fatal("decideCreate(missing character) error = nil, want failure")
	}

	createEvents, err := module.Decide(state, caller.MustNewAIAgent("agent-1"), command.Envelope{
		CampaignID: "camp-1",
		Message: scene.Create{
			Name:         " Opening ",
			CharacterIDs: []string{"char-1"},
		},
	}, staticSceneIDs("scene-1"))
	if err != nil {
		t.Fatalf("Decide(create) error = %v", err)
	}
	if got, want := len(createEvents), 1; got != want {
		t.Fatalf("create events len = %d, want %d", got, want)
	}
	created, err := event.MessageAs[scene.Created](createEvents[0])
	if err != nil {
		t.Fatalf("MessageAs(created) error = %v", err)
	}
	if got, want := created.SceneID, "scene-1"; got != want {
		t.Fatalf("created scene id = %q, want %q", got, want)
	}
	if err := module.Fold(&state, createEvents[0]); err != nil {
		t.Fatalf("Fold(created) error = %v", err)
	}
	if got, want := state.Scenes["scene-1"].SessionID, "sess-1"; got != want {
		t.Fatalf("scene session id = %q, want %q", got, want)
	}
	if err := module.Fold(nil, createEvents[0]); err == nil {
		t.Fatal("Fold(nil) error = nil, want failure")
	}
	if err := module.Fold(&state, event.Envelope{CampaignID: "camp-1", Message: unknownSceneEvent{}}); err == nil {
		t.Fatal("Fold(unknown) error = nil, want failure")
	}

	if _, err := decideActivate(state, command.Envelope{CampaignID: "camp-1", Message: scene.Activate{SceneID: "missing"}}); err == nil {
		t.Fatal("decideActivate(missing scene) error = nil, want failure")
	}
	activateEvents, err := module.Decide(state, caller.MustNewAIAgent("gm-1"), command.Envelope{
		CampaignID: "camp-1",
		Message:    scene.Activate{SceneID: "scene-1"},
	}, staticSceneIDs())
	if err != nil {
		t.Fatalf("Decide(activate) error = %v", err)
	}
	if err := module.Fold(&state, activateEvents[0]); err != nil {
		t.Fatalf("Fold(activated) error = %v", err)
	}
	if got, want := state.ActiveSceneID, "scene-1"; got != want {
		t.Fatalf("active scene id = %q, want %q", got, want)
	}

	if _, err := decideReplaceCast(state, command.Envelope{CampaignID: "camp-1", Message: scene.ReplaceCast{SceneID: "scene-1", CharacterIDs: []string{"missing"}}}); err == nil {
		t.Fatal("decideReplaceCast(missing character) error = nil, want failure")
	}
	replaceEvents, err := module.Decide(state, caller.MustNewAIAgent("gm-1"), command.Envelope{
		CampaignID: "camp-1",
		Message:    scene.ReplaceCast{SceneID: "scene-1", CharacterIDs: []string{"char-1", "char-2"}},
	}, staticSceneIDs())
	if err != nil {
		t.Fatalf("Decide(replace cast) error = %v", err)
	}
	if err := module.Fold(&state, replaceEvents[0]); err != nil {
		t.Fatalf("Fold(replace cast) error = %v", err)
	}
	if got, want := len(state.Scenes["scene-1"].CharacterIDs), 2; got != want {
		t.Fatalf("scene cast len = %d, want %d", got, want)
	}

	endEvents, err := module.Decide(state, caller.MustNewAIAgent("gm-1"), command.Envelope{
		CampaignID: "camp-1",
		Message:    scene.End{SceneID: "scene-1"},
	}, staticSceneIDs())
	if err != nil {
		t.Fatalf("Decide(end) error = %v", err)
	}
	if err := module.Fold(&state, endEvents[0]); err != nil {
		t.Fatalf("Fold(ended) error = %v", err)
	}
	if got := state.ActiveSceneID; got != "" {
		t.Fatalf("active scene id after end = %q, want empty", got)
	}
}

type testSceneCommand struct{}

func (testSceneCommand) CommandType() command.Type { return "test.command" }

type unknownSceneEvent struct{}

func (unknownSceneEvent) EventType() event.Type { return "test.unknown" }

func staticSceneIDs(ids ...string) func(string) (string, error) {
	index := 0
	return func(string) (string, error) {
		if index >= len(ids) {
			return "", errors.New("unexpected id allocation")
		}
		next := ids[index]
		index++
		return next, nil
	}
}

func readySceneCampaignState() campaign.State {
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.PlayState = campaign.PlayStateActive
	state.AIAgentID = "agent-1"
	state.Sessions["sess-1"] = session.Record{
		ID:     "sess-1",
		Name:   "Session 1",
		Status: session.StatusActive,
	}
	state.ActiveSessionID = "sess-1"
	state.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-owner",
		Active: true,
	}
	state.Participants["gm-1"] = participant.Record{
		ID:   "gm-1",
		Name: "Narrator", Access: participant.AccessMember, Active: true,
	}
	state.Characters["char-1"] = character.Record{
		ID:            "char-1",
		ParticipantID: "owner-1",
		Name:          "Luna", Active: true,
	}
	state.Characters["char-2"] = character.Record{
		ID:            "char-2",
		ParticipantID: "owner-1",
		Name:          "Iris", Active: true,
	}
	return state
}
