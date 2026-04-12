package participant

import (
	"errors"
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

func TestModuleMetadata(t *testing.T) {
	t.Parallel()

	module := New()
	if got, want := module.Name(), "core.participant"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if got, want := len(module.Commands()), 5; got != want {
		t.Fatalf("Commands() len = %d, want %d", got, want)
	}
	if got, want := len(module.Events()), 5; got != want {
		t.Fatalf("Events() len = %d, want %d", got, want)
	}

	registration := module.Commands()[0]
	if got, want := registration.Spec.Definition().Type, participant.CommandTypeJoin; got != want {
		t.Fatalf("command type = %s, want %s", got, want)
	}
	if registration.Admission.SupportsPlanning {
		t.Fatal("SupportsPlanning = true, want false")
	}
	if got, want := len(registration.Admission.AllowedPlayStates), 1; got != want {
		t.Fatalf("allowed play states len = %d, want %d", got, want)
	}
	if got, want := registration.Admission.AllowedPlayStates[0], campaign.PlayStateSetup; got != want {
		t.Fatalf("allowed play state = %q, want %q", got, want)
	}
	if err := registration.Admission.Authorize(caller.MustNewSubject("subject-2"), campaign.State{}); err == nil {
		t.Fatal("Authorize(join) error = nil, want failure")
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
	state.Participants["part-1"] = participant.Record{ID: "part-1", Name: "owner", Access: participant.AccessOwner, SubjectID: "subject-1", Active: true}

	if _, err := decideJoin(campaign.NewState(), caller.MustNewSubject("subject-2"), command.Envelope{CampaignID: "camp-1", Message: participant.Join{Name: "zoe", Access: participant.AccessMember}}, staticIDs("part-2")); err == nil {
		t.Fatal("decideJoin(missing campaign) error = nil, want failure")
	}
	if _, err := decideJoin(state, caller.MustNewSubject("subject-2"), command.Envelope{CampaignID: "camp-1", Message: participant.Join{Name: "zoe", Access: participant.AccessMember, SubjectID: "subject-1"}}, staticIDs("part-2")); err == nil {
		t.Fatal("decideJoin(duplicate subject) error = nil, want failure")
	}
	if _, err := decideJoin(state, caller.MustNewSubject("subject-2"), command.Envelope{CampaignID: "camp-1", Message: testCommand{}}, staticIDs("part-2")); err == nil {
		t.Fatal("decideJoin(bad message) error = nil, want failure")
	}
	if _, err := decideJoin(state, caller.MustNewSubject("subject-2"), command.Envelope{CampaignID: "camp-1", Message: participant.Join{Name: "zoe", Access: participant.AccessMember}}, staticIDs("   ")); err == nil {
		t.Fatal("decideJoin(invalid joined event) error = nil, want failure")
	}
	if _, err := module.Decide(state, caller.MustNewSubject("subject-2"), command.Envelope{CampaignID: "camp-1", Message: participant.Join{Name: "zoe", Access: participant.AccessMember}}, func(string) (string, error) {
		return "", errors.New("boom")
	}); err == nil {
		t.Fatal("Decide(join alloc) error = nil, want failure")
	}

	events, err := module.Decide(state, caller.MustNewSubject("subject-2"), command.Envelope{CampaignID: "camp-1", Message: participant.Join{Name: " zoe ", Access: participant.AccessMember}}, staticIDs("part-2"))
	if err != nil {
		t.Fatalf("Decide(join) error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
	joined, err := event.MessageAs[participant.Joined](events[0])
	if err != nil {
		t.Fatalf("MessageAs(joined) error = %v", err)
	}
	if got := joined.SubjectID; got != "" {
		t.Fatalf("joined subject = %q, want empty", got)
	}
	if err := module.Fold(&state, events[0]); err != nil {
		t.Fatalf("Fold(joined) error = %v", err)
	}
	if got, want := state.Participants["part-2"].Name, "zoe"; got != want {
		t.Fatalf("participant name = %q, want %q", got, want)
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
