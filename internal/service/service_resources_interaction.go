package service

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

type interactionResource struct {
	Interaction interactionPayload `json:"interaction"`
}

type interactionPayload struct {
	CampaignID    string             `json:"campaign_id"`
	PlayState     campaign.PlayState `json:"play_state"`
	ActiveSession *session.Record    `json:"active_session"`
	ActiveScene   *scene.Record      `json:"active_scene,omitempty"`
	Scenes        []scene.Record     `json:"scenes,omitempty"`
}

func (s *Service) readInteractionResource(request campaignResourceRequest) (any, error) {
	return interactionResource{
		Interaction: interactionPayload{
			CampaignID:    request.snapshot.ID,
			PlayState:     request.snapshot.PlayState,
			ActiveSession: request.snapshot.ActiveSession(),
			ActiveScene:   request.snapshot.ActiveScene(),
			Scenes:        append([]scene.Record(nil), request.snapshot.Scenes...),
		},
	}, nil
}
