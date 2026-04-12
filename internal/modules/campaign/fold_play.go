package campaign

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/event"
)

func foldPlayBegan(state *campaign.State, envelope event.Envelope) error {
	state.PlayState = campaign.PlayStateActive
	return nil
}

func foldPlayPaused(state *campaign.State, envelope event.Envelope) error {
	state.PlayState = campaign.PlayStatePaused
	return nil
}

func foldPlayResumed(state *campaign.State, envelope event.Envelope) error {
	state.PlayState = campaign.PlayStateActive
	return nil
}

func foldPlayEnded(state *campaign.State, envelope event.Envelope) error {
	state.PlayState = campaign.PlayStateSetup
	return nil
}
