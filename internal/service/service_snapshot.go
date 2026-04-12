package service

import (
	"context"
	"fmt"
)

func (s *Service) publishedCampaignSnapshot(ctx context.Context, campaignID string) (publishedCampaignSnapshot, error) {
	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return publishedCampaignSnapshot{}, err
	}
	slot, release := s.acquireCampaignSlot(campaignID)
	defer release()
	if snapshot, ok := slot.loadPublished(); ok {
		return snapshot, nil
	}

	slot.mu.Lock()
	defer slot.mu.Unlock()

	if snapshot, ok := slot.loadPublished(); ok {
		return snapshot, nil
	}

	timeline, _, err := s.loadCampaignInSlot(ctx, slot, campaignID)
	if err != nil {
		return publishedCampaignSnapshot{}, err
	}
	snapshot, ok := slot.loadPublished()
	if ok {
		return snapshot, nil
	}
	return publishedCampaignSnapshot{}, fmt.Errorf(
		"published snapshot missing after load for campaign %s at head seq %d",
		campaignID,
		headSeqOf(timeline),
	)
}
