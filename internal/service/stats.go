package service

import "time"

// Stats exposes lightweight internal diagnostics for one service instance.
type Stats struct {
	CampaignSlots          int
	PublishedSnapshots     int
	WarmRuntimes           int
	OldestPublishedAt      time.Time
	OldestRuntimeTouchedAt time.Time
}

// Stats returns internal slot, published snapshot, and runtime-cache counts.
func (s *Service) Stats() Stats {
	if s == nil || s.slots == nil {
		return Stats{}
	}

	now := time.Time{}
	if s.recordClock != nil {
		now = s.recordClock.Now()
	}

	slots := s.slots.snapshot()
	stats := Stats{CampaignSlots: len(slots)}
	for _, slot := range slots {
		if slot == nil {
			continue
		}

		if snapshot := slot.published.Load(); snapshot != nil {
			stats.PublishedSnapshots++
			if stats.OldestPublishedAt.IsZero() || snapshot.touchedAt.Before(stats.OldestPublishedAt) {
				stats.OldestPublishedAt = snapshot.touchedAt
			}
		}

		slot.mu.Lock()
		runtime := slot.runtime
		slot.mu.Unlock()
		if runtime == nil {
			continue
		}
		if s.runtimeTTL > 0 && !now.IsZero() && now.Sub(runtime.touchedAt) >= s.runtimeTTL {
			continue
		}
		stats.WarmRuntimes++
		if stats.OldestRuntimeTouchedAt.IsZero() || runtime.touchedAt.Before(stats.OldestRuntimeTouchedAt) {
			stats.OldestRuntimeTouchedAt = runtime.touchedAt
		}
	}
	return stats
}

func (r *campaignSlotRegistry) snapshot() []*campaignSlot {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*campaignSlot, 0, len(r.slots))
	for _, slot := range r.slots {
		out = append(out, slot)
	}
	return out
}
