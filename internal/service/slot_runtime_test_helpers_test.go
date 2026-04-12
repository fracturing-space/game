package service

import "github.com/fracturing-space/game/internal/campaign"

func testLoadRuntime(svc *Service, campaignID string, headSeq uint64) (campaign.State, bool) {
	if svc == nil {
		return campaign.State{}, false
	}
	slot := svc.campaignSlot(campaignID)
	slot.mu.Lock()
	defer slot.mu.Unlock()
	return slot.loadRuntime(headSeq, svc.recordClock.Now(), svc.runtimeTTL)
}
