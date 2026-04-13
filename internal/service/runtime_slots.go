package service

import (
	"sync"
	"time"
)

type campaignSlotRegistry struct {
	mu    sync.Mutex
	slots map[string]*campaignSlot
}

type campaignSlot struct {
	mu        sync.Mutex
	runtime   *campaignRuntime
	published atomicPublishedCampaignSnapshot
	refs      int
	touchedAt time.Time
}

func newCampaignSlotRegistry() *campaignSlotRegistry {
	return &campaignSlotRegistry{slots: make(map[string]*campaignSlot)}
}

func (r *campaignSlotRegistry) Slot(campaignID string) *campaignSlot {
	if r == nil {
		return &campaignSlot{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if slot, ok := r.slots[campaignID]; ok {
		return slot
	}
	slot := &campaignSlot{}
	r.slots[campaignID] = slot
	return slot
}

func (r *campaignSlotRegistry) Acquire(campaignID string, now time.Time, idleTTL time.Duration) *campaignSlot {
	if r == nil {
		return &campaignSlot{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.sweepIdleLocked(now, idleTTL)

	slot, ok := r.slots[campaignID]
	if !ok || slot == nil {
		slot = &campaignSlot{}
		r.slots[campaignID] = slot
	}
	slot.refs++
	slot.touch(now)
	return slot
}

func (r *campaignSlotRegistry) Release(campaignID string, slot *campaignSlot, now time.Time, idleTTL time.Duration) {
	if r == nil || slot == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.slots[campaignID]
	if ok && current == slot {
		if current.refs > 0 {
			current.refs--
		}
		current.touch(now)
	}
	r.sweepIdleLocked(now, idleTTL)
}

func (s *Service) campaignSlot(campaignID string) *campaignSlot {
	if s == nil || s.slots == nil {
		return &campaignSlot{}
	}
	return s.slots.Slot(campaignID)
}

func (s *Service) acquireCampaignSlot(campaignID string) (*campaignSlot, func()) {
	if s == nil || s.slots == nil {
		return &campaignSlot{}, func() {}
	}

	now := s.slotAccessTime()
	slot := s.slots.Acquire(campaignID, now, s.slotIdleTTL)
	return slot, func() {
		s.slots.Release(campaignID, slot, s.slotAccessTime(), s.slotIdleTTL)
	}
}

func (s *Service) slotAccessTime() time.Time {
	if s == nil || s.recordClock == nil {
		return time.Time{}
	}
	return s.recordClock.Now()
}

func (slot *campaignSlot) touch(now time.Time) {
	if slot == nil || now.IsZero() {
		return
	}
	if now.After(slot.touchedAt) {
		slot.touchedAt = now
	}
}

func (slot *campaignSlot) lastTouchedAt() time.Time {
	if slot == nil {
		return time.Time{}
	}
	return slot.touchedAt
}

func (r *campaignSlotRegistry) sweepIdleLocked(now time.Time, idleTTL time.Duration) {
	if r == nil || idleTTL <= 0 || now.IsZero() {
		return
	}

	for campaignID, slot := range r.slots {
		if slot == nil || slot.refs != 0 {
			continue
		}
		lastTouchedAt := slot.lastTouchedAt()
		if lastTouchedAt.IsZero() {
			continue
		}
		if now.Sub(lastTouchedAt) >= idleTTL {
			delete(r.slots, campaignID)
		}
	}
}
