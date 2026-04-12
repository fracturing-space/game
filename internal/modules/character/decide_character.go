package character

import (
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
)

func decideCreate(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[character.Create](envelope)
	if err != nil {
		return nil, err
	}
	if err := authorizeCharacterWrite(state, act, message.ParticipantID, "", true); err != nil {
		return nil, err
	}
	if _, err := requireActiveParticipant(state, message.ParticipantID); err != nil {
		return nil, err
	}
	characterID, err := ids("char")
	if err != nil {
		return nil, err
	}
	created, err := event.NewEnvelope(
		character.CreatedEventSpec,
		envelope.CampaignID,
		character.Created{
			CharacterID:   characterID,
			ParticipantID: message.ParticipantID,
			Name:          message.Name,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{created}, nil
}

func decideUpdate(state campaign.State, act caller.Caller, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[character.Update](envelope)
	if err != nil {
		return nil, err
	}
	record, err := requireActiveCharacter(state, message.CharacterID)
	if err != nil {
		return nil, err
	}
	if _, err := requireActiveParticipant(state, message.ParticipantID); err != nil {
		return nil, err
	}
	if err := authorizeCharacterWrite(state, act, message.ParticipantID, record.ParticipantID, false); err != nil {
		return nil, err
	}
	updated, err := event.NewEnvelope(
		character.UpdatedEventSpec,
		envelope.CampaignID,
		character.Updated{
			CharacterID:   record.ID,
			ParticipantID: message.ParticipantID,
			Name:          message.Name,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{updated}, nil
}

func decideDelete(state campaign.State, act caller.Caller, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[character.Delete](envelope)
	if err != nil {
		return nil, err
	}
	record, err := requireActiveCharacter(state, message.CharacterID)
	if err != nil {
		return nil, err
	}
	if err := authorizeCharacterWrite(state, act, record.ParticipantID, record.ParticipantID, false); err != nil {
		return nil, err
	}
	if characterReferencedByActiveSession(state, record.ID) {
		return nil, errs.FailedPreconditionf("character %s is still referenced by the active session", record.ID)
	}
	if characterReferencedByActiveScene(state, record.ID) {
		return nil, errs.FailedPreconditionf("character %s is still referenced by the active scene", record.ID)
	}
	deleted, err := event.NewEnvelope(
		character.DeletedEventSpec,
		envelope.CampaignID,
		character.Deleted{CharacterID: record.ID},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{deleted}, nil
}
