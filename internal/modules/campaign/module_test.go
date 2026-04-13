package campaign

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
	if got, want := module.Name(), "core.campaign"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	if got, want := len(module.Commands()), 8; got != want {
		t.Fatalf("Commands() len = %d, want %d", got, want)
	}
	if got, want := len(module.Events()), 8; got != want {
		t.Fatalf("Events() len = %d, want %d", got, want)
	}
}

func TestModuleCommandsAdmissions(t *testing.T) {
	t.Parallel()

	module := New()
	state := readyState()
	regs := module.Commands()
	if got, want := regs[0].Spec.Definition().Type, campaign.CommandTypeCreate; got != want {
		t.Fatalf("first command type = %q, want %q", got, want)
	}
	if len(regs[0].Admission.AllowedPlayStates) != 0 {
		t.Fatalf("create admission play states = %v, want unrestricted", regs[0].Admission.AllowedPlayStates)
	}
	if err := regs[0].Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("create authorize(valid) error = %v", err)
	}
	for _, reg := range regs[1:] {
		if err := reg.Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err != nil {
			t.Fatalf("%s authorize(owner) error = %v", reg.Spec.Definition().Type, err)
		}
	}
}

func TestDecideCreateCreatesOnlyOwnerParticipant(t *testing.T) {
	t.Parallel()

	module := New()
	events, err := module.Decide(campaign.NewState(), caller.MustNewSubject("subject-owner"), command.Envelope{
		Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "Louis"},
	}, staticIDs("camp-1", "part-1"))
	if err != nil {
		t.Fatalf("Decide(create) error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
	joined, err := event.MessageAs[participant.Joined](events[1])
	if err != nil {
		t.Fatalf("MessageAs(joined) error = %v", err)
	}
	if got, want := joined.Access, participant.AccessOwner; got != want {
		t.Fatalf("owner access = %q, want %q", got, want)
	}
}

func TestDecidePlayBeginAutoStartsSession(t *testing.T) {
	t.Parallel()

	module := New()
	state := readyState()

	events, err := module.Decide(state, caller.MustNewSubject("subject-owner"), command.Envelope{
		CampaignID: "camp-1",
		Message:    campaign.PlayBegin{},
	}, staticIDs("sess-1"))
	if err != nil {
		t.Fatalf("Decide(play begin) error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
	if _, err := event.MessageAs[campaign.PlayBegan](events[1]); err != nil {
		t.Fatalf("MessageAs(play began) error = %v", err)
	}
}

func TestModuleFoldBranches(t *testing.T) {
	t.Parallel()

	module := New()
	if err := module.Fold(nil, event.Envelope{}); err == nil {
		t.Fatal("Fold(nil state) error = nil, want failure")
	}

	state := campaign.NewState()
	cases := []event.Envelope{
		mustCampaignEvent(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{Name: "Autumn Twilight"}),
		mustCampaignEvent(t, campaign.UpdatedEventSpec, "camp-1", campaign.Updated{Name: "Autumn Dusk"}),
		mustCampaignEvent(t, campaign.AIBoundEventSpec, "camp-1", campaign.AIBound{AIAgentID: "agent-1"}),
		mustCampaignEvent(t, campaign.AIUnboundEventSpec, "camp-1", campaign.AIUnbound{}),
		mustCampaignEvent(t, campaign.PlayBeganEventSpec, "camp-1", campaign.PlayBegan{SessionID: "sess-1"}),
		mustCampaignEvent(t, campaign.PlayPausedEventSpec, "camp-1", campaign.PlayPaused{SessionID: "sess-1", SceneID: "scene-1"}),
		mustCampaignEvent(t, campaign.PlayResumedEventSpec, "camp-1", campaign.PlayResumed{SessionID: "sess-1", SceneID: "scene-1"}),
		mustCampaignEvent(t, campaign.PlayEndedEventSpec, "camp-1", campaign.PlayEnded{SessionID: "sess-1"}),
	}
	for _, envelope := range cases {
		if err := module.Fold(&state, envelope); err != nil {
			t.Fatalf("Fold(%s) error = %v", envelope.Type(), err)
		}
	}
	if !state.Exists || state.CampaignID != "camp-1" || state.Name != "Autumn Dusk" || state.PlayState != campaign.PlayStateSetup {
		t.Fatalf("Fold() final state = %+v, want updated campaign state", state)
	}
	if err := module.Fold(&state, mustParticipantEvent(t)); err == nil {
		t.Fatal("Fold(unknown event) error = nil, want failure")
	}
}

func TestDecideUpdateAndAIBindingBranches(t *testing.T) {
	t.Parallel()

	state := readyState()

	if _, err := decideUpdate(campaign.NewState(), command.Envelope{CampaignID: "camp-1", Message: campaign.Update{Name: "Autumn Dusk"}}); err == nil {
		t.Fatal("decideUpdate(missing campaign) error = nil, want failure")
	}
	events, err := decideUpdate(state, command.Envelope{CampaignID: "camp-1", Message: campaign.Update{Name: "Autumn Dusk"}})
	if err != nil {
		t.Fatalf("decideUpdate(valid) error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("decideUpdate events len = %d, want %d", got, want)
	}

	if _, err := decideAIBind(campaign.NewState(), command.Envelope{CampaignID: "camp-1", Message: campaign.AIBind{AIAgentID: "agent-1"}}); err == nil {
		t.Fatal("decideAIBind(missing campaign) error = nil, want failure")
	}
	events, err = decideAIBind(state, command.Envelope{CampaignID: "camp-1", Message: campaign.AIBind{AIAgentID: "agent-1"}})
	if err != nil {
		t.Fatalf("decideAIBind(valid) error = %v", err)
	}
	if _, err := event.MessageAs[campaign.AIBound](events[0]); err != nil {
		t.Fatalf("decideAIBind event = %v, want ai bound", err)
	}

	if _, err := decideAIUnbind(campaign.NewState(), command.Envelope{CampaignID: "camp-1", Message: campaign.AIUnbind{}}); err == nil {
		t.Fatal("decideAIUnbind(missing campaign) error = nil, want failure")
	}
	events, err = decideAIUnbind(state, command.Envelope{CampaignID: "camp-1", Message: campaign.AIUnbind{}})
	if err != nil {
		t.Fatalf("decideAIUnbind(valid) error = %v", err)
	}
	if _, err := event.MessageAs[campaign.AIUnbound](events[0]); err != nil {
		t.Fatalf("decideAIUnbind event = %v, want ai unbound", err)
	}
}

func TestDecidePlayEndResumeAndSessionHelperBranches(t *testing.T) {
	t.Parallel()

	state := readyState()
	state.PlayState = campaign.PlayStatePaused
	state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
	state.ActiveSessionID = "sess-1"
	state.ActiveSceneID = "scene-1"

	events, err := decidePlayResume(state, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayResume{Reason: "resume"}})
	if err != nil {
		t.Fatalf("decidePlayResume(valid) error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("decidePlayResume events len = %d, want %d", got, want)
	}

	missingSession := readyState()
	missingSession.PlayState = campaign.PlayStatePaused
	if _, err := decidePlayResume(missingSession, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayResume{}}); err == nil {
		t.Fatal("decidePlayResume(no session) error = nil, want failure")
	}
	missingScene := state
	missingScene.ActiveSceneID = ""
	if _, err := decidePlayResume(missingScene, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayResume{}}); err == nil {
		t.Fatal("decidePlayResume(no scene) error = nil, want failure")
	}

	active := readyState()
	active.PlayState = campaign.PlayStateActive
	active.Sessions["sess-1"] = session.Record{
		ID:                   "sess-1",
		Name:                 "Session 1",
		Status:               session.StatusActive,
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}
	active.ActiveSessionID = "sess-1"
	if _, err := decidePlayEnd(active, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayEnd{}}); err != nil {
		t.Fatalf("decidePlayEnd(valid) error = %v", err)
	}
	active.ActiveSessionID = ""
	if _, err := decidePlayEnd(active, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayEnd{}}); err == nil {
		t.Fatal("decidePlayEnd(no session) error = nil, want failure")
	}

	if _, _, err := newSessionStartedEvents(readyState(), command.Envelope{CampaignID: "camp-1", Message: campaign.PlayBegin{}}, func(string) (string, error) {
		return "", errors.New("boom")
	}); err == nil {
		t.Fatal("newSessionStartedEvents(id failure) error = nil, want failure")
	}

	broken := readyState()
	broken.Characters["char-2"] = character.Record{ID: "char-2", Name: "Ghost", Active: true}
	if _, _, err := newSessionStartedEvents(broken, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayBegin{}}, staticIDs("sess-1")); err == nil {
		t.Fatal("newSessionStartedEvents(incomplete defaults) error = nil, want failure")
	}
}

func readyState() campaign.State {
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.PlayState = campaign.PlayStateSetup
	state.AIAgentID = "agent-1"
	state.SessionCount = 0
	state.Participants["part-1"] = participant.Record{
		ID:        "part-1",
		Name:      "Owner",
		Access:    participant.AccessOwner,
		SubjectID: "subject-owner",
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
		value := values[index]
		index++
		return value, nil
	}
}

func mustCampaignEvent[M event.Message](t *testing.T, spec event.TypedSpec[M], campaignID string, message M) event.Envelope {
	t.Helper()
	envelope, err := event.NewEnvelope(spec, campaignID, message)
	if err != nil {
		t.Fatalf("NewEnvelope(%s) error = %v", message.EventType(), err)
	}
	return envelope
}

func mustParticipantEvent(t *testing.T) event.Envelope {
	t.Helper()
	envelope, err := event.NewEnvelope(participant.JoinedEventSpec, "camp-1", participant.Joined{
		ParticipantID: "part-1",
		Name:          "Owner",
		Access:        participant.AccessOwner,
		SubjectID:     "subject-owner",
	})
	if err != nil {
		t.Fatalf("NewEnvelope(participant.joined) error = %v", err)
	}
	return envelope
}
