package participant

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

func foldJoined(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[participant.Joined](envelope)
	if err != nil {
		return err
	}
	state.Participants[message.ParticipantID] = participant.Record{
		ID:        message.ParticipantID,
		Name:      message.Name,
		Access:    message.Access,
		SubjectID: message.SubjectID,
		Active:    true,
	}
	return nil
}

func foldUpdated(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[participant.Updated](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Participants[message.ParticipantID]
	if !ok {
		return errs.NotFoundf("participant %s not found", message.ParticipantID)
	}
	record.Name = message.Name
	record.Access = message.Access
	state.Participants[message.ParticipantID] = record
	return nil
}

func foldBound(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[participant.Bound](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Participants[message.ParticipantID]
	if !ok {
		return errs.NotFoundf("participant %s not found", message.ParticipantID)
	}
	record.SubjectID = message.SubjectID
	state.Participants[message.ParticipantID] = record
	return nil
}

func foldUnbound(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[participant.Unbound](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Participants[message.ParticipantID]
	if !ok {
		return errs.NotFoundf("participant %s not found", message.ParticipantID)
	}
	record.SubjectID = ""
	state.Participants[message.ParticipantID] = record
	return nil
}

func foldLeft(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[participant.Left](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Participants[message.ParticipantID]
	if !ok {
		return errs.NotFoundf("participant %s not found", message.ParticipantID)
	}
	record.Active = false
	record.SubjectID = ""
	state.Participants[message.ParticipantID] = record
	return nil
}
