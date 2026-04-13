package character

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
)

func foldCreated(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[character.Created](envelope)
	if err != nil {
		return err
	}
	state.Characters[message.CharacterID] = character.Record{
		ID:            message.CharacterID,
		ParticipantID: message.ParticipantID,
		Name:          message.Name,
		Active:        true,
	}
	return nil
}

func foldUpdated(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[character.Updated](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Characters[message.CharacterID]
	if !ok {
		return errs.NotFoundf("character %s not found", message.CharacterID)
	}
	record.ParticipantID = message.ParticipantID
	record.Name = message.Name
	state.Characters[message.CharacterID] = record
	return nil
}

func foldDeleted(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[character.Deleted](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Characters[message.CharacterID]
	if !ok {
		return errs.NotFoundf("character %s not found", message.CharacterID)
	}
	record.Active = false
	state.Characters[message.CharacterID] = record
	return nil
}
