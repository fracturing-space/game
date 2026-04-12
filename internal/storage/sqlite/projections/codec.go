package projections

import (
	"bytes"
	"encoding/gob"

	"github.com/fracturing-space/game/internal/campaign"
)

func encodeCampaignState(state campaign.State) ([]byte, error) {
	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(state); err != nil {
		return nil, err
	}
	return payload.Bytes(), nil
}

func decodeCampaignState(payload []byte) (campaign.State, error) {
	state := campaign.NewState()
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&state); err != nil {
		return campaign.State{}, err
	}
	return state, nil
}
