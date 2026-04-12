package session

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/modules/util"
	"github.com/fracturing-space/game/internal/session"
)

func decideStart(state campaign.State, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	if state.ActiveSession() != nil {
		return nil, errs.AlreadyExistsf("campaign %s already has an active session", state.CampaignID)
	}
	message, err := command.MessageAs[session.Start](envelope)
	if err != nil {
		return nil, err
	}

	sessionID, err := ids("sess")
	if err != nil {
		return nil, err
	}
	started, _, err := util.BuildPlayStartEvent(state, envelope.CampaignID, sessionID, message.Name, message.CharacterControllers)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{started}, nil
}

func decideEnd(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	return util.BuildPlayEndEvents(state, envelope.CampaignID)
}
