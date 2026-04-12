package character

import (
	"slices"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/participant"
)

func authorizeCharacterWrite(state campaign.State, act caller.Caller, targetParticipantID string, existingParticipantID string, allowOwnerTransfer bool) error {
	if owner, ok := campaign.BoundParticipant(state, act.SubjectID); ok && owner.Access == participant.AccessOwner {
		return nil
	}
	record, ok := campaign.BoundParticipant(state, act.SubjectID)
	if !ok {
		return authz.Denied(authz.CapabilityCreateCharacter, "participant binding is required")
	}
	if targetParticipantID != record.ID {
		return authz.Denied(authz.CapabilityUpdateCharacter, "participant may only manage their own characters")
	}
	if !allowOwnerTransfer && existingParticipantID != "" && existingParticipantID != record.ID {
		return authz.Denied(authz.CapabilityUpdateCharacter, "participant may only manage their own characters")
	}
	return nil
}

func requireActiveCharacter(state campaign.State, characterID string) (character.Record, error) {
	record, ok := state.Characters[characterID]
	if !ok || !record.Active {
		return character.Record{}, errs.NotFoundf("character %s not found", characterID)
	}
	return record, nil
}

func requireActiveParticipant(state campaign.State, participantID string) (campaignParticipant, error) {
	record, ok := state.Participants[participantID]
	if !ok || !record.Active {
		return campaignParticipant{}, errs.NotFoundf("participant %s not found", participantID)
	}
	return campaignParticipant{ID: record.ID}, nil
}

type campaignParticipant struct {
	ID string
}

func characterReferencedByActiveSession(state campaign.State, characterID string) bool {
	activeSession := state.ActiveSession()
	if activeSession == nil {
		return false
	}
	for _, assignment := range activeSession.CharacterControllers {
		if assignment.CharacterID == characterID {
			return true
		}
	}
	return false
}

func characterReferencedByActiveScene(state campaign.State, characterID string) bool {
	for _, record := range state.Scenes {
		if record.ID != state.ActiveSceneID || record.Ended {
			continue
		}
		if slices.Contains(record.CharacterIDs, characterID) {
			return true
		}
	}
	return false
}
