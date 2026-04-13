package campaign

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/event"
)

func foldCreated(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[campaign.Created](envelope)
	if err != nil {
		return err
	}
	state.Exists = true
	state.CampaignID = envelope.CampaignID
	state.Name = message.Name
	state.PlayState = campaign.PlayStateSetup
	return nil
}

func foldUpdated(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[campaign.Updated](envelope)
	if err != nil {
		return err
	}
	state.Name = message.Name
	return nil
}

func foldAIBound(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[campaign.AIBound](envelope)
	if err != nil {
		return err
	}
	state.AIAgentID = message.AIAgentID
	return nil
}

func foldAIUnbound(state *campaign.State, envelope event.Envelope) error {
	state.AIAgentID = ""
	return nil
}
