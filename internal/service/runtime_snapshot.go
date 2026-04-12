package service

import (
	"sync/atomic"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
)

type campaignRuntime struct {
	headSeq   uint64
	state     campaign.State
	touchedAt time.Time
}

type publishedCampaignSnapshot struct {
	headSeq   uint64
	state     campaign.State
	touchedAt time.Time
}

type atomicPublishedCampaignSnapshot struct {
	ptr atomic.Pointer[publishedCampaignSnapshot]
}

func (s *atomicPublishedCampaignSnapshot) Load() *publishedCampaignSnapshot {
	if s == nil {
		return nil
	}
	return s.ptr.Load()
}

func (s *atomicPublishedCampaignSnapshot) Store(snapshot *publishedCampaignSnapshot) {
	if s == nil {
		return
	}
	s.ptr.Store(snapshot)
}

func (slot *campaignSlot) loadRuntime(headSeq uint64, now time.Time, ttl time.Duration) (campaign.State, bool) {
	if slot == nil || slot.runtime == nil {
		return campaign.State{}, false
	}
	if ttl > 0 && now.Sub(slot.runtime.touchedAt) >= ttl {
		slot.runtime = nil
		return campaign.State{}, false
	}
	if slot.runtime.headSeq != headSeq {
		return campaign.State{}, false
	}
	slot.runtime.touchedAt = now
	slot.touch(now)
	return slot.runtime.state.Clone(), true
}

func (slot *campaignSlot) storeRuntime(headSeq uint64, state campaign.State, touchedAt time.Time) {
	if slot == nil {
		return
	}
	slot.runtime = &campaignRuntime{
		headSeq:   headSeq,
		state:     state.Clone(),
		touchedAt: touchedAt,
	}
	slot.touch(touchedAt)
}

func (slot *campaignSlot) loadPublished() (publishedCampaignSnapshot, bool) {
	if slot == nil {
		return publishedCampaignSnapshot{}, false
	}
	snapshot := slot.published.Load()
	if snapshot == nil {
		return publishedCampaignSnapshot{}, false
	}
	return publishedCampaignSnapshot{
		headSeq:   snapshot.headSeq,
		state:     snapshot.state.Clone(),
		touchedAt: snapshot.touchedAt,
	}, true
}

func (slot *campaignSlot) storePublished(headSeq uint64, state campaign.State, touchedAt time.Time) {
	if slot == nil {
		return
	}
	slot.published.Store(&publishedCampaignSnapshot{
		headSeq:   headSeq,
		state:     state.Clone(),
		touchedAt: touchedAt,
	})
	slot.touch(touchedAt)
}
