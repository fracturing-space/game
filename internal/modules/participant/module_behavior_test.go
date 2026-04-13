package participant

import (
	"testing"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/session"
)

func TestModuleParticipantLifecyclePaths(t *testing.T) {
	t.Parallel()

	module := New()
	state := participantModuleState()

	if _, err := decideUpdate(campaign.NewState(), command.Envelope{CampaignID: "camp-1", Message: participant.Update{ParticipantID: "part-2"}}); err == nil {
		t.Fatal("decideUpdate(missing campaign) error = nil, want failure")
	}
	if _, err := decideUpdate(state, command.Envelope{CampaignID: "camp-1", Message: testCommand{}}); err == nil {
		t.Fatal("decideUpdate(bad message) error = nil, want failure")
	}
	if _, err := decideUpdate(state, command.Envelope{CampaignID: "camp-1", Message: participant.Update{
		ParticipantID: "owner-1",
		Name:          "Owner", Access: participant.AccessOwner}}); err == nil {
		t.Fatal("decideUpdate(owner reassigned) error = nil, want failure")
	}
	if _, err := decideUpdate(state, command.Envelope{CampaignID: "camp-1", Message: participant.Update{
		ParticipantID: "part-2",
		Name:          "Player", Access: participant.AccessOwner}}); err == nil {
		t.Fatal("decideUpdate(owner transfer) error = nil, want failure")
	}
	updateEvents, err := module.Decide(state, caller.MustNewSubject("subject-owner"), command.Envelope{
		CampaignID: "camp-1",
		Message: participant.Update{
			ParticipantID: "part-2",
			Name:          " Player Two ", Access: participant.AccessMember},
	}, staticIDs())
	if err != nil {
		t.Fatalf("Decide(update) error = %v", err)
	}
	if err := module.Fold(&state, updateEvents[0]); err != nil {
		t.Fatalf("Fold(updated) error = %v", err)
	}
	if got, want := state.Participants["part-2"].Name, "Player Two"; got != want {
		t.Fatalf("participant name = %q, want %q", got, want)
	}
	invalidUpdatedState := participantModuleState()
	invalidUpdatedState.Participants[""] = participant.Record{
		ID:   "",
		Name: "Nameless", Access: participant.AccessMember, SubjectID: "subject-nameless",
		Active: true,
	}
	if _, err := decideUpdate(invalidUpdatedState, command.Envelope{CampaignID: "camp-1", Message: participant.Update{
		ParticipantID: "",
		Name:          "Nameless", Access: participant.AccessMember,
	}}); err == nil {
		t.Fatal("decideUpdate(invalid updated event) error = nil, want failure")
	}

	if _, err := decideBind(state, caller.MustNewSubject("subject-other"), command.Envelope{CampaignID: "camp-1", Message: participant.Bind{ParticipantID: "part-3"}}); err == nil {
		t.Fatal("decideBind(already bound by other subject) error = nil, want failure")
	}
	if _, err := decideBind(campaign.NewState(), caller.MustNewSubject("subject-guest"), command.Envelope{CampaignID: "camp-1", Message: participant.Bind{ParticipantID: "part-3"}}); err == nil {
		t.Fatal("decideBind(missing campaign) error = nil, want failure")
	}
	if _, err := decideBind(state, caller.MustNewSubject("subject-guest"), command.Envelope{CampaignID: "camp-1", Message: testCommand{}}); err == nil {
		t.Fatal("decideBind(bad message) error = nil, want failure")
	}
	unboundState := participantModuleState()
	unboundState.Participants["part-3"] = participant.Record{
		ID:   "part-3",
		Name: "Guest", Access: participant.AccessMember, Active: true,
	}
	bindEvents, err := module.Decide(unboundState, caller.MustNewSubject("subject-guest"), command.Envelope{
		CampaignID: "camp-1",
		Message:    participant.Bind{ParticipantID: "part-3"},
	}, staticIDs())
	if err != nil {
		t.Fatalf("Decide(bind) error = %v", err)
	}
	bound, err := event.MessageAs[participant.Bound](bindEvents[0])
	if err != nil {
		t.Fatalf("MessageAs(bound) error = %v", err)
	}
	if got, want := bound.SubjectID, "subject-guest"; got != want {
		t.Fatalf("bound subject id = %q, want %q", got, want)
	}
	if err := module.Fold(&unboundState, bindEvents[0]); err != nil {
		t.Fatalf("Fold(bound) error = %v", err)
	}
	if got, want := unboundState.Participants["part-3"].SubjectID, "subject-guest"; got != want {
		t.Fatalf("folded subject id = %q, want %q", got, want)
	}
	invalidBoundState := participantModuleState()
	invalidBoundState.Participants[""] = participant.Record{
		ID:   "",
		Name: "Guest", Access: participant.AccessMember, Active: true,
	}
	if _, err := decideBind(invalidBoundState, caller.MustNewSubject("subject-guest"), command.Envelope{
		CampaignID: "camp-1",
		Message:    participant.Bind{ParticipantID: ""},
	}); err == nil {
		t.Fatal("decideBind(invalid bound event) error = nil, want failure")
	}

	if _, err := decideBind(state, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: participant.Bind{ParticipantID: "ai-gm"}}); err == nil {
		t.Fatal("decideBind(non-human participant) error = nil, want failure")
	}
	if _, err := decideBind(unboundState, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: participant.Bind{ParticipantID: "part-3"}}); err == nil {
		t.Fatal("decideBind(subject already bound in campaign) error = nil, want failure")
	}

	if _, err := decideUnbind(state, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: participant.Unbind{ParticipantID: "part-4"}}); err == nil {
		t.Fatal("decideUnbind(not bound participant) error = nil, want failure")
	}
	if _, err := decideUnbind(campaign.NewState(), caller.MustNewSubject("subject-member"), command.Envelope{CampaignID: "camp-1", Message: participant.Unbind{ParticipantID: "part-2"}}); err == nil {
		t.Fatal("decideUnbind(missing campaign) error = nil, want failure")
	}
	if _, err := decideUnbind(state, caller.MustNewSubject("subject-member"), command.Envelope{CampaignID: "camp-1", Message: testCommand{}}); err == nil {
		t.Fatal("decideUnbind(bad message) error = nil, want failure")
	}
	if _, err := decideUnbind(state, caller.MustNewSubject("subject-member"), command.Envelope{CampaignID: "camp-1", Message: participant.Unbind{ParticipantID: "owner-1"}}); err == nil {
		t.Fatal("decideUnbind(owner participant) error = nil, want failure")
	}
	if _, err := decideUnbind(state, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: participant.Unbind{ParticipantID: "ai-gm"}}); err == nil {
		t.Fatal("decideUnbind(non-human participant) error = nil, want failure")
	}
	unbindEvents, err := module.Decide(state, caller.MustNewSubject("subject-member"), command.Envelope{
		CampaignID: "camp-1",
		Message:    participant.Unbind{ParticipantID: "part-2"},
	}, staticIDs())
	if err != nil {
		t.Fatalf("Decide(unbind) error = %v", err)
	}
	if err := module.Fold(&state, unbindEvents[0]); err != nil {
		t.Fatalf("Fold(unbound) error = %v", err)
	}
	if got := state.Participants["part-2"].SubjectID; got != "" {
		t.Fatalf("participant subject id after unbind = %q, want empty", got)
	}
	invalidUnboundState := participantModuleState()
	invalidUnboundState.Participants[""] = participant.Record{
		ID:   "",
		Name: "Bound", Access: participant.AccessMember, SubjectID: "subject-blank",
		Active: true,
	}
	if _, err := decideUnbind(invalidUnboundState, caller.MustNewSubject("subject-blank"), command.Envelope{
		CampaignID: "camp-1",
		Message:    participant.Unbind{ParticipantID: ""},
	}); err == nil {
		t.Fatal("decideUnbind(invalid unbound event) error = nil, want failure")
	}

	leaveState := participantModuleState()
	leaveState.Participants["part-5"] = participant.Record{
		ID:   "part-5",
		Name: "Leaving", Access: participant.AccessMember, SubjectID: "subject-leaving",
		Active: true,
	}
	leaveState.Characters["char-5"] = character.Record{
		ID:            "char-5",
		ParticipantID: "part-5",
		Name:          "Luna", Active: true,
	}
	if _, err := decideLeave(leaveState, command.Envelope{CampaignID: "camp-1", Message: participant.Leave{ParticipantID: "owner-1"}}); err == nil {
		t.Fatal("decideLeave(owner participant) error = nil, want failure")
	}
	if _, err := decideLeave(campaign.NewState(), command.Envelope{CampaignID: "camp-1", Message: participant.Leave{ParticipantID: "part-5"}}); err == nil {
		t.Fatal("decideLeave(missing campaign) error = nil, want failure")
	}
	if _, err := decideLeave(leaveState, command.Envelope{CampaignID: "camp-1", Message: testCommand{}}); err == nil {
		t.Fatal("decideLeave(bad message) error = nil, want failure")
	}
	if _, err := decideLeave(leaveState, command.Envelope{CampaignID: "camp-1", Message: participant.Leave{ParticipantID: "part-5"}}); err == nil {
		t.Fatal("decideLeave(active characters) error = nil, want failure")
	}
	delete(leaveState.Characters, "char-5")
	leaveState.Sessions["sess-1"] = session.Record{
		ID:                   "sess-1",
		Name:                 "Session 1",
		Status:               session.StatusActive,
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-5"}},
	}
	leaveState.ActiveSessionID = "sess-1"
	if _, err := decideLeave(leaveState, command.Envelope{CampaignID: "camp-1", Message: participant.Leave{ParticipantID: "part-5"}}); err == nil {
		t.Fatal("decideLeave(active controller) error = nil, want failure")
	}
	leaveState.ActiveSessionID = ""
	leaveEvents, err := module.Decide(leaveState, caller.MustNewSubject("subject-owner"), command.Envelope{
		CampaignID: "camp-1",
		Message:    participant.Leave{ParticipantID: "part-5"},
	}, staticIDs())
	if err != nil {
		t.Fatalf("Decide(leave) error = %v", err)
	}
	if err := module.Fold(&leaveState, leaveEvents[0]); err != nil {
		t.Fatalf("Fold(left) error = %v", err)
	}
	if leaveState.Participants["part-5"].Active {
		t.Fatal("participant still active after leave")
	}
	invalidLeftState := participantModuleState()
	invalidLeftState.Participants[""] = participant.Record{
		ID:   "",
		Name: "Leaving", Access: participant.AccessMember, Active: true,
	}
	if _, err := decideLeave(invalidLeftState, command.Envelope{
		CampaignID: "camp-1",
		Message:    participant.Leave{ParticipantID: ""},
	}); err == nil {
		t.Fatal("decideLeave(invalid left event) error = nil, want failure")
	}

	if _, err := requireActiveParticipant(state, "missing"); err == nil {
		t.Fatal("requireActiveParticipant(missing) error = nil, want failure")
	}
	state.Participants["inactive"] = participant.Record{ID: "inactive"}
	if _, err := requireActiveParticipant(state, "inactive"); err == nil {
		t.Fatal("requireActiveParticipant(inactive) error = nil, want failure")
	}
	if err := authorizeSeatManagement(state, caller.MustNewSubject("subject-other"), state.Participants["part-2"], true, authz.CapabilityUnbindParticipant); err == nil {
		t.Fatal("authorizeSeatManagement(other participant) error = nil, want failure")
	}

	commands := module.Commands()
	if got, want := len(commands), 5; got != want {
		t.Fatalf("commands len = %d, want %d", got, want)
	}
	if err := commands[1].Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("update participant authorize error = %v", err)
	}
	if err := commands[2].Admission.Authorize(caller.MustNewSubject("subject-member"), state); err != nil {
		t.Fatalf("bind participant authorize error = %v", err)
	}
	if err := commands[3].Admission.Authorize(caller.MustNewSubject("subject-member"), state); err != nil {
		t.Fatalf("unbind participant authorize error = %v", err)
	}
	if err := commands[4].Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("delete participant authorize error = %v", err)
	}

	foldState := participantModuleState()
	foldState.Participants["part-6"] = participant.Record{
		ID:   "part-6",
		Name: "Folded", Access: participant.AccessMember, SubjectID: "subject-old",
		Active: true,
	}
	if err := module.Fold(&foldState, mustEnvelope(t, participant.UpdatedEventSpec, "camp-1", participant.Updated{
		ParticipantID: "part-6",
		Name:          "Updated", Access: participant.AccessMember})); err != nil {
		t.Fatalf("Fold(updated) extra error = %v", err)
	}
	if err := module.Fold(&foldState, mustEnvelope(t, participant.BoundEventSpec, "camp-1", participant.Bound{
		ParticipantID: "part-6",
		SubjectID:     "subject-new",
	})); err != nil {
		t.Fatalf("Fold(bound) extra error = %v", err)
	}
	if err := module.Fold(&foldState, mustEnvelope(t, participant.UnboundEventSpec, "camp-1", participant.Unbound{
		ParticipantID: "part-6",
	})); err != nil {
		t.Fatalf("Fold(unbound) extra error = %v", err)
	}
	if err := module.Fold(&foldState, mustEnvelope(t, participant.LeftEventSpec, "camp-1", participant.Left{
		ParticipantID: "part-6",
	})); err != nil {
		t.Fatalf("Fold(left) extra error = %v", err)
	}
}

func participantModuleState() campaign.State {
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.PlayState = campaign.PlayStateSetup
	state.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-owner",
		Active: true,
	}
	state.Participants["part-2"] = participant.Record{
		ID:   "part-2",
		Name: "Player", Access: participant.AccessMember, SubjectID: "subject-member",
		Active: true,
	}
	state.Participants["ai-gm"] = participant.Record{
		ID:   "ai-gm",
		Name: "Narrator", Access: participant.AccessMember, Active: true,
	}
	state.Participants["part-4"] = participant.Record{
		ID:   "part-4",
		Name: "Guest", Access: participant.AccessMember, Active: true,
	}
	return state
}

func mustEnvelope[T event.Message](t *testing.T, spec event.TypedSpec[T], campaignID string, message T) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(spec, campaignID, message)
	if err != nil {
		t.Fatalf("event.NewEnvelope() error = %v", err)
	}
	return envelope
}
