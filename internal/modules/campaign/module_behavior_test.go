package campaign

import (
	"testing"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
)

func TestPlayLifecycleBranches(t *testing.T) {
	t.Parallel()

	state := readyState()

	if _, err := decidePlayPause(state, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayPause{}}); err == nil {
		t.Fatal("decidePlayPause(setup) error = nil, want failure")
	}

	state.PlayState = campaign.PlayStateActive
	state.ActiveSessionID = ""
	if _, err := decidePlayPause(state, command.Envelope{CampaignID: "camp-1", Message: campaign.PlayPause{}}); err == nil {
		t.Fatal("decidePlayPause(no session) error = nil, want failure")
	}
}
