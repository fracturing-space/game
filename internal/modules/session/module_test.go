package session

import (
	"errors"
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/session"
)

func TestModuleMetadata(t *testing.T) {
	t.Parallel()

	module := New()
	if got, want := module.Name(), "core.session"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if got, want := len(module.Commands()), 2; got != want {
		t.Fatalf("Commands() len = %d, want %d", got, want)
	}
	if got, want := len(module.Events()), 2; got != want {
		t.Fatalf("Events() len = %d, want %d", got, want)
	}
}

func TestModuleCommandsAdmissions(t *testing.T) {
	t.Parallel()

	module := New()
	state := readyCampaignState()
	regs := module.Commands()
	if got, want := regs[0].Spec.Definition().Type, session.CommandTypeStart; got != want {
		t.Fatalf("first command type = %q, want %q", got, want)
	}
	for _, reg := range regs {
		if err := reg.Admission.Authorize(caller.MustNewSubject("subject-1"), state); err != nil {
			t.Fatalf("%s authorize(owner) error = %v", reg.Spec.Definition().Type, err)
		}
	}
}

func TestModuleDecideStartAndFold(t *testing.T) {
	t.Parallel()

	module := New()
	state := readyCampaignState()

	events, err := module.Decide(state, caller.MustNewSubject("subject-1"), command.Envelope{
		CampaignID: "camp-1",
		Message: session.Start{
			Name: " Night Watch ",
			CharacterControllers: []session.CharacterControllerAssignment{{
				CharacterID:   "char-1",
				ParticipantID: "part-1",
			}},
		},
	}, staticIDs("sess-1"))
	if err != nil {
		t.Fatalf("Decide(start) error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
	started, err := event.MessageAs[session.Started](events[0])
	if err != nil {
		t.Fatalf("MessageAs(started) error = %v", err)
	}
	if got, want := started.Name, "Night Watch"; got != want {
		t.Fatalf("started name = %q, want %q", got, want)
	}

	if err := module.Fold(&state, events[0]); err != nil {
		t.Fatalf("Fold(started) error = %v", err)
	}
	if state.ActiveSession() == nil {
		t.Fatal("active session = nil, want session")
	}
}

func TestModuleFoldEndedAndErrors(t *testing.T) {
	t.Parallel()

	module := New()
	if err := module.Fold(nil, event.Envelope{}); err == nil {
		t.Fatal("Fold(nil state) error = nil, want failure")
	}

	state := readyCampaignState()
	state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Night Watch", Status: session.StatusActive}
	state.ActiveSessionID = "sess-1"
	state.ActiveSceneID = "scene-1"
	ended, err := event.NewEnvelope(session.EndedEventSpec, "camp-1", session.Ended{
		SessionID:            "sess-1",
		Name:                 "Night Watch",
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	})
	if err != nil {
		t.Fatalf("NewEnvelope(ended) error = %v", err)
	}
	if err := module.Fold(&state, ended); err != nil {
		t.Fatalf("Fold(ended) error = %v", err)
	}
	if state.ActiveSession() != nil || state.ActiveSceneID != "" {
		t.Fatalf("Fold(ended) state = %+v, want cleared session and scene", state)
	}
	if err := module.Fold(&state, mustUnknownSessionEvent(t)); err == nil {
		t.Fatal("Fold(unknown event) error = nil, want failure")
	}
}

func TestModuleDecideEnd(t *testing.T) {
	t.Parallel()

	module := New()
	state := readyCampaignState()
	state.PlayState = campaign.PlayStateActive
	state.Sessions["sess-1"] = session.Record{
		ID:                   "sess-1",
		Name:                 "Night Watch",
		Status:               session.StatusActive,
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}
	state.ActiveSessionID = "sess-1"

	events, err := module.Decide(state, caller.MustNewSubject("subject-1"), command.Envelope{
		CampaignID: "camp-1",
		Message:    session.End{},
	}, staticIDs())
	if err != nil {
		t.Fatalf("Decide(end) error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
}

func TestModuleErrors(t *testing.T) {
	t.Parallel()

	module := New()
	if _, err := module.Decide(campaign.NewState(), caller.MustNewSubject("subject-1"), command.Envelope{Message: testCommand{}}, staticIDs()); err == nil {
		t.Fatal("Decide(unknown) error = nil, want failure")
	}

	state := readyCampaignState()
	state.Characters["char-empty"] = character.Record{ID: "char-empty", Name: "Ghost", Active: true}
	if _, err := decideStart(state, command.Envelope{CampaignID: "camp-1", Message: session.Start{}}, staticIDs("sess-1")); err == nil {
		t.Fatal("decideStart(incomplete defaults) error = nil, want failure")
	}
	if _, err := decideStart(readyCampaignState(), command.Envelope{CampaignID: "camp-1", Message: session.Start{}}, func(string) (string, error) {
		return "", errors.New("boom")
	}); err == nil {
		t.Fatal("decideStart(id alloc) error = nil, want failure")
	}

	state.PlayState = campaign.PlayStateSetup
	if _, err := decideEnd(state, command.Envelope{CampaignID: "camp-1", Message: session.End{}}); err == nil {
		t.Fatal("decideEnd(no active session) error = nil, want failure")
	}
	if _, err := decideEnd(campaign.NewState(), command.Envelope{CampaignID: "camp-1", Message: session.End{}}); err == nil {
		t.Fatal("decideEnd(missing campaign) error = nil, want failure")
	}
}

func readyCampaignState() campaign.State {
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Participants["part-1"] = participant.Record{
		ID:        "part-1",
		Name:      "Owner",
		Access:    participant.AccessOwner,
		SubjectID: "subject-1",
		Active:    true,
	}
	state.Characters["char-1"] = character.Record{
		ID:            "char-1",
		ParticipantID: "part-1",
		Name:          "Luna",
		Active:        true,
	}
	return state
}

func staticIDs(values ...string) func(string) (string, error) {
	index := 0
	return func(string) (string, error) {
		if index >= len(values) {
			return "", nil
		}
		value := values[index]
		index++
		return value, nil
	}
}

type testCommand struct{}

func (testCommand) CommandType() command.Type { return "test.command" }

func mustUnknownSessionEvent(t *testing.T) event.Envelope {
	t.Helper()
	envelope, err := event.NewEnvelope(campaign.PlayEndedEventSpec, "camp-1", campaign.PlayEnded{SessionID: "sess-1"})
	if err != nil {
		t.Fatalf("NewEnvelope(play ended) error = %v", err)
	}
	return envelope
}
