package campaign

import (
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

func decideCreate(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	if state.Exists {
		return nil, errs.AlreadyExistsf("campaign already exists")
	}
	message, err := command.MessageAs[campaign.Create](envelope)
	if err != nil {
		return nil, err
	}

	campaignID, err := ids("camp")
	if err != nil {
		return nil, err
	}
	created, err := event.NewEnvelope(
		campaign.CreatedEventSpec,
		campaignID,
		campaign.Created{Name: message.Name},
	)
	if err != nil {
		return nil, err
	}
	participantID, err := ids("part")
	if err != nil {
		return nil, err
	}
	joined, err := event.NewEnvelope(
		participant.JoinedEventSpec,
		campaignID,
		participant.Joined{
			ParticipantID: participantID,
			Name:          message.OwnerName,
			Access:        participant.AccessOwner,
			SubjectID:     act.SubjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{created, joined}, nil
}

func decideUpdate(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[campaign.Update](envelope)
	if err != nil {
		return nil, err
	}
	updated, err := event.NewEnvelope(
		campaign.UpdatedEventSpec,
		envelope.CampaignID,
		campaign.Updated(message),
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{updated}, nil
}

func decideAIBind(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[campaign.AIBind](envelope)
	if err != nil {
		return nil, err
	}
	bound, err := event.NewEnvelope(
		campaign.AIBoundEventSpec,
		envelope.CampaignID,
		campaign.AIBound(message),
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{bound}, nil
}

func decideAIUnbind(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	unbound, err := event.NewEnvelope(
		campaign.AIUnboundEventSpec,
		envelope.CampaignID,
		campaign.AIUnbound{},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{unbound}, nil
}
