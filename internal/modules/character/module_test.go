package character

import (
	"errors"
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

func TestModuleMetadata(t *testing.T) {
	t.Parallel()

	module := New()
	if got, want := module.Name(), "core.character"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if got, want := len(module.Commands()), 3; got != want {
		t.Fatalf("Commands() len = %d, want %d", got, want)
	}
	if got, want := len(module.Events()), 3; got != want {
		t.Fatalf("Events() len = %d, want %d", got, want)
	}
	commands := module.Commands()
	if got, want := len(commands[0].Admission.AllowedPlayStates), 1; got != want {
		t.Fatalf("AllowedPlayStates len = %d, want %d", got, want)
	}
	if got, want := commands[0].Admission.AllowedPlayStates[0], campaign.PlayStateSetup; got != want {
		t.Fatalf("AllowedPlayStates[0] = %q, want %q", got, want)
	}
	if err := commands[0].Admission.Authorize(caller.MustNewSubject("subject-1"), campaign.State{}); err != nil {
		t.Fatalf("Admission.Authorize() error = %v", err)
	}
	if got, want := commands[0].Spec.Definition().Type, character.CommandTypeCreate; got != want {
		t.Fatalf("command spec type = %q, want %q", got, want)
	}
	if got, want := module.Events()[0].Definition().Type, character.EventTypeCreated; got != want {
		t.Fatalf("event spec type = %q, want %q", got, want)
	}
}

func TestModuleDecideAndFold(t *testing.T) {
	t.Parallel()

	module := New()
	if _, err := module.Decide(campaign.NewState(), caller.MustNewSubject("subject-1"), command.Envelope{Message: testCommand{}}, staticIDs()); err == nil {
		t.Fatal("Decide(unknown) error = nil, want failure")
	}

	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.PlayState = campaign.PlayStateSetup
	state.Participants["part-1"] = participant.Record{
		ID:     "part-1",
		Name:   "owner",
		Access: participant.AccessOwner, SubjectID: "subject-1",
		Active: true,
	}

	if _, err := decideCreate(campaign.NewState(), caller.MustNewSubject("subject-1"), command.Envelope{CampaignID: "camp-1", Message: character.Create{ParticipantID: "part-1", Name: "luna"}}, staticIDs("char-1")); err == nil {
		t.Fatal("decideCreate(missing campaign) error = nil, want failure")
	}
	if _, err := decideCreate(state, caller.MustNewSubject("subject-2"), command.Envelope{CampaignID: "camp-1", Message: character.Create{ParticipantID: "part-1", Name: "luna"}}, staticIDs("char-1")); err == nil {
		t.Fatal("decideCreate(unbound caller) error = nil, want failure")
	}
	if _, err := decideCreate(state, caller.MustNewSubject("subject-1"), command.Envelope{CampaignID: "camp-1", Message: testCommand{}}, staticIDs("char-1")); err == nil {
		t.Fatal("decideCreate(bad message) error = nil, want failure")
	}
	if _, err := decideCreate(state, caller.MustNewSubject("subject-1"), command.Envelope{CampaignID: "camp-1", Message: character.Create{ParticipantID: "part-1", Name: "luna"}}, staticIDs("   ")); err == nil {
		t.Fatal("decideCreate(invalid created event) error = nil, want failure")
	}
	state.Participants["part-2"] = participant.Record{
		ID:     "part-2",
		Name:   "guest",
		Access: participant.AccessMember, Active: true,
	}
	if _, err := decideCreate(state, caller.Caller{}, command.Envelope{CampaignID: "camp-1", Message: character.Create{ParticipantID: "part-1", Name: "luna"}}, staticIDs("char-1")); err == nil {
		t.Fatal("decideCreate(empty subject) error = nil, want failure")
	}
	if _, err := module.Decide(state, caller.MustNewSubject("subject-1"), command.Envelope{CampaignID: "camp-1", Message: character.Create{ParticipantID: "part-1", Name: "luna"}}, func(string) (string, error) {
		return "", errors.New("boom")
	}); err == nil {
		t.Fatal("Decide(create alloc) error = nil, want failure")
	}

	events, err := module.Decide(state, caller.MustNewSubject("subject-1"), command.Envelope{CampaignID: "camp-1", Message: character.Create{ParticipantID: "part-1", Name: " luna "}}, staticIDs("char-1"))
	if err != nil {
		t.Fatalf("Decide(create) error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
	created, err := event.MessageAs[character.Created](events[0])
	if err != nil {
		t.Fatalf("MessageAs(created) error = %v", err)
	}
	if got, want := created.ParticipantID, "part-1"; got != want {
		t.Fatalf("created participant id = %q, want %q", got, want)
	}
	if got, want := created.Name, "luna"; got != want {
		t.Fatalf("created name = %q, want %q", got, want)
	}
	if err := module.Fold(&state, events[0]); err != nil {
		t.Fatalf("Fold(created) error = %v", err)
	}
	if got, want := state.Characters["char-1"].Name, "luna"; got != want {
		t.Fatalf("character name = %q, want %q", got, want)
	}
	if got, want := state.Characters["char-1"].ParticipantID, "part-1"; got != want {
		t.Fatalf("character participant id = %q, want %q", got, want)
	}
	if err := module.Fold(nil, events[0]); err == nil {
		t.Fatal("Fold(nil) error = nil, want failure")
	}
	if err := module.Fold(&state, event.Envelope{CampaignID: "camp-1", Message: unknownEvent{}}); err == nil {
		t.Fatal("Fold(unknown) error = nil, want failure")
	}
}

type testCommand struct{}

func (testCommand) CommandType() command.Type { return "test.command" }

type unknownEvent struct{}

func (unknownEvent) EventType() event.Type { return "test.unknown" }

func staticIDs(ids ...string) func(string) (string, error) {
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
