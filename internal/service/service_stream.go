package service

import (
	"context"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
)

// SubscribeEvents returns an authorized committed-event feed for one campaign.
// Authorization is checked when the stream opens and is not re-evaluated after
// subscription setup.
func (s *Service) SubscribeEvents(ctx context.Context, act caller.Caller, campaignID string, afterSeq uint64) (EventStream, error) {
	if err := ctx.Err(); err != nil {
		return EventStream{}, err
	}

	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return EventStream{}, err
	}
	snapshot, err := s.publishedCampaignSnapshot(ctx, campaignID)
	if err != nil {
		return EventStream{}, err
	}
	if err := authz.RequireReadCampaign(act, snapshot.state.Clone()); err != nil {
		return EventStream{}, err
	}
	subscription, err := s.store.SubscribeAfter(ctx, campaignID, afterSeq)
	if err != nil {
		return EventStream{}, err
	}
	return EventStream(subscription), nil
}
