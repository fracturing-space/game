package service

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/readiness"
)

func (s *Service) loadCampaignLocked(ctx context.Context, campaignID string) ([]event.Record, campaign.State, error) {
	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return nil, campaign.State{}, err
	}
	slot, release := s.acquireCampaignSlot(campaignID)
	defer release()
	slot.mu.Lock()
	defer slot.mu.Unlock()
	return s.loadCampaignInSlot(ctx, slot, campaignID)
}

func (s *Service) loadCampaignInSlot(ctx context.Context, slot *campaignSlot, campaignID string) ([]event.Record, campaign.State, error) {
	timeline, ok, err := s.store.List(ctx, campaignID)
	if err != nil {
		return nil, campaign.State{}, err
	}
	if !ok {
		return nil, campaign.State{}, errs.NotFoundf("campaign %s not found", campaignID)
	}
	headSeq := headSeqOf(timeline)
	if state, ok := slot.loadRuntime(headSeq, s.recordClock.Now(), s.runtimeTTL); ok {
		slot.storePublished(headSeq, state, s.recordClock.Now())
		return timeline, state, nil
	}
	if projection, ok, err := s.projections.GetProjection(ctx, campaignID); err != nil {
		return nil, campaign.State{}, err
	} else if ok && projection.HeadSeq == headSeq {
		slot.storeRuntime(headSeq, projection.State, s.recordClock.Now())
		slot.storePublished(headSeq, projection.State, projection.UpdatedAt)
		return timeline, projection.State.Clone(), nil
	}

	state, err := s.replayLocked(timeline)
	if err != nil {
		return nil, campaign.State{}, err
	}
	if err := s.persistProjectionInSlot(ctx, slot, campaignID, state, headSeq, lastActivityAtOfTimeline(timeline)); err != nil {
		return nil, campaign.State{}, err
	}
	return timeline, state, nil
}

func (s *Service) persistProjectionLocked(ctx context.Context, campaignID string, state campaign.State, headSeq uint64, lastActivityAt time.Time) error {
	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return err
	}
	slot, release := s.acquireCampaignSlot(campaignID)
	defer release()
	slot.mu.Lock()
	defer slot.mu.Unlock()
	return s.persistProjectionInSlot(ctx, slot, campaignID, state, headSeq, lastActivityAt)
}

func (s *Service) persistProjectionInSlot(ctx context.Context, slot *campaignSlot, campaignID string, state campaign.State, headSeq uint64, lastActivityAt time.Time) error {
	now := s.recordClock.Now()
	if lastActivityAt.IsZero() {
		lastActivityAt = now
	}
	snapshot := ProjectionSnapshot{
		CampaignID:     campaignID,
		HeadSeq:        headSeq,
		State:          state,
		UpdatedAt:      now,
		LastActivityAt: lastActivityAt,
	}
	watermark := ProjectionWatermark{
		CampaignID:      campaignID,
		AppliedSeq:      headSeq,
		ExpectedNextSeq: headSeq + 1,
		UpdatedAt:       now,
	}
	if err := s.projections.SaveProjectionAndWatermark(ctx, snapshot, watermark); err != nil {
		return err
	}
	slot.storePublished(headSeq, state, now)
	slot.storeRuntime(headSeq, state, now)
	return nil
}

func headSeqOf(timeline []event.Record) uint64 {
	if len(timeline) == 0 {
		return 0
	}
	return timeline[len(timeline)-1].Seq
}

func staleProjectionGap(snapshot ProjectionSnapshot, headSeq uint64) bool {
	return snapshot.HeadSeq != headSeq
}

func projectionLag(watermark ProjectionWatermark, headSeq uint64) uint64 {
	if headSeq <= watermark.AppliedSeq {
		return 0
	}
	return headSeq - watermark.AppliedSeq
}

func projectionUpdatedAt(snapshot ProjectionSnapshot, watermark ProjectionWatermark) time.Time {
	if watermark.UpdatedAt.After(snapshot.UpdatedAt) {
		return watermark.UpdatedAt
	}
	return snapshot.UpdatedAt
}

func lastActivityAtOfTimeline(timeline []event.Record) time.Time {
	if len(timeline) == 0 {
		return time.Time{}
	}
	return timeline[len(timeline)-1].RecordedAt.UTC()
}

func CampaignSummaryFromSnapshot(snapshot ProjectionSnapshot) CampaignSummary {
	state := snapshot.State.Clone()
	report := readiness.EvaluatePlay(state)
	return CampaignSummary{
		CampaignID:       snapshot.CampaignID,
		Name:             state.Name,
		ReadyToPlay:      report.Ready(),
		HasAIBinding:     campaign.HasBoundAIAgent(state),
		HasActiveSession: state.ActiveSession() != nil,
		LastActivityAt:   snapshot.LastActivityAt.UTC(),
	}
}

func BoundSubjectIDs(state campaign.State) []string {
	seen := make(map[string]struct{})
	subjectIDs := make([]string, 0, len(state.Participants))
	for _, record := range state.Participants {
		subjectID := record.SubjectID
		if subjectID == "" {
			continue
		}
		if _, ok := seen[subjectID]; ok {
			continue
		}
		seen[subjectID] = struct{}{}
		subjectIDs = append(subjectIDs, subjectID)
	}
	slices.Sort(subjectIDs)
	return subjectIDs
}

func CompareCampaignSummary(a CampaignSummary, b CampaignSummary) int {
	switch {
	case a.LastActivityAt.After(b.LastActivityAt):
		return -1
	case a.LastActivityAt.Before(b.LastActivityAt):
		return 1
	default:
		return strings.Compare(a.CampaignID, b.CampaignID)
	}
}
