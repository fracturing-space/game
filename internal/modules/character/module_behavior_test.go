package character

import (
	"testing"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestModuleCharacterUpdateDeleteAndHelpers(t *testing.T) {
	t.Parallel()

	module := New()
	state := characterModuleState()

	commands := module.Commands()
	if got, want := len(commands), 3; got != want {
		t.Fatalf("commands len = %d, want %d", got, want)
	}
	if err := commands[0].Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("create character authorize error = %v", err)
	}
	if err := commands[1].Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("update character authorize error = %v", err)
	}
	if err := commands[2].Admission.Authorize(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("delete character authorize error = %v", err)
	}

	if _, err := decideUpdate(campaign.NewState(), caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: character.Update{CharacterID: "char-1"}}); err == nil {
		t.Fatal("decideUpdate(missing campaign) error = nil, want failure")
	}
	if _, err := decideUpdate(state, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: testCommand{}}); err == nil {
		t.Fatal("decideUpdate(bad message) error = nil, want failure")
	}
	if _, err := decideUpdate(state, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: character.Update{
		CharacterID:   "char-1",
		ParticipantID: "missing",
		Name:          "Nova"}}); err == nil {
		t.Fatal("decideUpdate(missing participant) error = nil, want failure")
	}
	if _, err := decideDelete(campaign.NewState(), caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: character.Delete{CharacterID: "char-1"}}); err == nil {
		t.Fatal("decideDelete(missing campaign) error = nil, want failure")
	}
	if _, err := decideDelete(state, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: testCommand{}}); err == nil {
		t.Fatal("decideDelete(bad message) error = nil, want failure")
	}
	updateEvents, err := module.Decide(state, caller.MustNewSubject("subject-owner"), command.Envelope{
		CampaignID: "camp-1",
		Message: character.Update{
			CharacterID:   "char-1",
			ParticipantID: "part-1",
			Name:          " Nova "},
	}, staticIDs())
	if err != nil {
		t.Fatalf("Decide(update) error = %v", err)
	}
	updated, err := event.MessageAs[character.Updated](updateEvents[0])
	if err != nil {
		t.Fatalf("MessageAs(updated) error = %v", err)
	}
	if got, want := updated.Name, "Nova"; got != want {
		t.Fatalf("updated name = %q, want %q", got, want)
	}
	if err := module.Fold(&state, updateEvents[0]); err != nil {
		t.Fatalf("Fold(updated) error = %v", err)
	}
	if got, want := state.Characters["char-1"].Name, "Nova"; got != want {
		t.Fatalf("state character name = %q, want %q", got, want)
	}

	playerState := characterModuleState()
	playerState.Participants["part-1"] = participant.Record{
		ID:   "part-1",
		Name: "Player", Access: participant.AccessMember, SubjectID: "subject-player",
		Active: true,
	}
	playerState.Participants["part-2"] = participant.Record{
		ID:   "part-2",
		Name: "Other", Access: participant.AccessMember, Active: true,
	}
	if err := authorizeCharacterWrite(playerState, caller.MustNewSubject("subject-missing"), "part-1", "", true); err == nil {
		t.Fatal("authorizeCharacterWrite(unbound) error = nil, want failure")
	}
	if err := authorizeCharacterWrite(playerState, caller.MustNewSubject("subject-player"), "part-2", "", true); err == nil {
		t.Fatal("authorizeCharacterWrite(other owner) error = nil, want failure")
	}
	if err := authorizeCharacterWrite(playerState, caller.MustNewSubject("subject-player"), "part-1", "other-owner", false); err == nil {
		t.Fatal("authorizeCharacterWrite(existing owner mismatch) error = nil, want failure")
	}
	if err := authorizeCharacterWrite(state, caller.MustNewSubject("subject-owner"), "part-1", "", true); err != nil {
		t.Fatalf("authorizeCharacterWrite(owner) error = %v", err)
	}

	deleteState := characterModuleState()
	deleteState.Sessions["sess-1"] = session.Record{
		ID:                   "sess-1",
		Name:                 "Session 1",
		Status:               session.StatusActive,
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}
	deleteState.ActiveSessionID = "sess-1"
	if _, err := decideDelete(deleteState, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: character.Delete{CharacterID: "char-1"}}); err == nil {
		t.Fatal("decideDelete(active session reference) error = nil, want failure")
	}
	deleteState.ActiveSessionID = ""
	deleteState.ActiveSceneID = "scene-1"
	deleteState.Scenes["scene-1"] = scene.Record{
		ID:           "scene-1",
		SessionID:    "sess-1",
		Name:         "Opening",
		Active:       true,
		CharacterIDs: []string{"char-1"},
	}
	if _, err := decideDelete(deleteState, caller.MustNewSubject("subject-owner"), command.Envelope{CampaignID: "camp-1", Message: character.Delete{CharacterID: "char-1"}}); err == nil {
		t.Fatal("decideDelete(active scene reference) error = nil, want failure")
	}
	delete(deleteState.Scenes, "scene-1")
	deleteState.ActiveSceneID = ""
	deleteEvents, err := module.Decide(deleteState, caller.MustNewSubject("subject-owner"), command.Envelope{
		CampaignID: "camp-1",
		Message:    character.Delete{CharacterID: "char-1"},
	}, staticIDs())
	if err != nil {
		t.Fatalf("Decide(delete) error = %v", err)
	}
	if err := module.Fold(&deleteState, deleteEvents[0]); err != nil {
		t.Fatalf("Fold(deleted) error = %v", err)
	}
	if deleteState.Characters["char-1"].Active {
		t.Fatal("character still active after delete")
	}

	if _, err := requireActiveCharacter(state, "missing"); err == nil {
		t.Fatal("requireActiveCharacter(missing) error = nil, want failure")
	}
	state.Characters["inactive"] = character.Record{ID: "inactive"}
	if _, err := requireActiveCharacter(state, "inactive"); err == nil {
		t.Fatal("requireActiveCharacter(inactive) error = nil, want failure")
	}
	if _, err := requireActiveParticipant(state, "missing"); err == nil {
		t.Fatal("requireActiveParticipant(missing) error = nil, want failure")
	}
	if !characterReferencedByActiveSession(characterModuleStateWithSession("char-1"), "char-1") {
		t.Fatal("characterReferencedByActiveSession() = false, want true")
	}
	if !characterReferencedByActiveScene(characterModuleStateWithScene("char-1"), "char-1") {
		t.Fatal("characterReferencedByActiveScene() = false, want true")
	}
	if err := authorizeCharacterWrite(playerState, caller.MustNewSubject("subject-player"), "part-1", "part-1", false); err != nil {
		t.Fatalf("authorizeCharacterWrite(player own pc) error = %v", err)
	}
	if !authz.IsDenied(authz.RequireUpdateCharacter(caller.Caller{})) {
		t.Fatal("sanity check: RequireUpdateCharacter should deny empty caller")
	}
	if characterReferencedByActiveSession(characterModuleState(), "missing") {
		t.Fatal("characterReferencedByActiveSession(missing) = true, want false")
	}
	if characterReferencedByActiveScene(characterModuleState(), "missing") {
		t.Fatal("characterReferencedByActiveScene(missing) = true, want false")
	}
}

func characterModuleState() campaign.State {
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.PlayState = campaign.PlayStateSetup
	state.Participants["part-1"] = participant.Record{
		ID:   "part-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-owner",
		Active: true,
	}
	state.Participants["part-2"] = participant.Record{
		ID:   "part-2",
		Name: "Player", Access: participant.AccessMember, SubjectID: "subject-player",
		Active: true,
	}
	state.Characters["char-1"] = character.Record{
		ID:            "char-1",
		ParticipantID: "part-1",
		Name:          "Luna", Active: true,
	}
	return state
}

func characterModuleStateWithSession(characterID string) campaign.State {
	state := characterModuleState()
	state.Sessions["sess-1"] = session.Record{
		ID:                   "sess-1",
		Name:                 "Session 1",
		Status:               session.StatusActive,
		CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: characterID, ParticipantID: "part-1"}},
	}
	state.ActiveSessionID = "sess-1"
	return state
}

func characterModuleStateWithScene(characterID string) campaign.State {
	state := characterModuleState()
	state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
	state.ActiveSessionID = "sess-1"
	state.ActiveSceneID = "scene-1"
	state.Scenes["scene-1"] = scene.Record{
		ID:           "scene-1",
		SessionID:    "sess-1",
		Name:         "Opening",
		Active:       true,
		CharacterIDs: []string{characterID},
	}
	return state
}
