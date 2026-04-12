package service

import (
	"context"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/readiness"
)

const listCampaignsLimit = 10

// Inspect replays one stored campaign timeline and returns the derived live
// state.
func (s *Service) Inspect(ctx context.Context, act caller.Caller, campaignID string) (Inspection, error) {
	if err := ctx.Err(); err != nil {
		return Inspection{}, err
	}

	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return Inspection{}, err
	}
	snapshot, err := s.publishedCampaignSnapshot(ctx, campaignID)
	if err != nil {
		return Inspection{}, err
	}
	state := snapshot.state.Clone()
	if err := authz.RequireReadCampaign(act, state); err != nil {
		return Inspection{}, err
	}

	timeline, _, err := s.store.ListAfter(ctx, campaignID, 0)
	if err != nil {
		return Inspection{}, err
	}
	timeline = recordsAtOrBeforeSeq(timeline, snapshot.headSeq)
	return Inspection{
		Timeline: cloneEventRecords(timeline),
		State:    campaign.SnapshotOf(state),
		HeadSeq:  snapshot.headSeq,
	}, nil
}

func (s *Service) readAuthorizedCampaignLocked(ctx context.Context, campaignID string, authorize func(campaign.State) error) (string, []event.Record, campaign.State, error) {
	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return "", nil, campaign.State{}, err
	}
	snapshot, err := s.publishedCampaignSnapshot(ctx, campaignID)
	if err != nil {
		return "", nil, campaign.State{}, err
	}
	state := snapshot.state.Clone()
	if authorize != nil {
		if err := authorize(state); err != nil {
			return "", nil, campaign.State{}, err
		}
	}
	timeline, _, err := s.store.ListAfter(ctx, campaignID, 0)
	if err != nil {
		return "", nil, campaign.State{}, err
	}
	return campaignID, recordsAtOrBeforeSeq(timeline, snapshot.headSeq), state, nil
}

// GetPlayReadiness returns deterministic blockers preventing entry into PLAY.
func (s *Service) GetPlayReadiness(ctx context.Context, act caller.Caller, campaignID string) (PlayReadiness, error) {
	if err := ctx.Err(); err != nil {
		return PlayReadiness{}, err
	}

	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return PlayReadiness{}, err
	}
	snapshot, err := s.publishedCampaignSnapshot(ctx, campaignID)
	if err != nil {
		return PlayReadiness{}, err
	}
	state := snapshot.state.Clone()
	if err := authz.RequireReadCampaign(act, state); err != nil {
		return PlayReadiness{}, err
	}

	report := readiness.EvaluatePlay(state)
	if err := readiness.ValidateReport(report); err != nil {
		return PlayReadiness{}, err
	}
	return report, nil
}

// ListEvents returns persisted campaign events after the supplied sequence.
func (s *Service) ListEvents(ctx context.Context, act caller.Caller, campaignID string, afterSeq uint64) ([]event.Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.publishedCampaignSnapshot(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := authz.RequireReadCampaign(act, snapshot.state.Clone()); err != nil {
		return nil, err
	}

	timeline, _, err := s.store.ListAfter(ctx, campaignID, afterSeq)
	if err != nil {
		return nil, err
	}
	return cloneEventRecords(recordsAtOrBeforeSeq(timeline, snapshot.headSeq)), nil
}

func (s *Service) readAuthorizedCampaignInSlot(ctx context.Context, slot *campaignSlot, campaignID string, authorize func(campaign.State) error) (string, []event.Record, campaign.State, error) {
	campaignID, err := normalizeCampaignID(campaignID)
	if err != nil {
		return "", nil, campaign.State{}, err
	}
	timeline, state, err := s.loadCampaignInSlot(ctx, slot, campaignID)
	if err != nil {
		return "", nil, campaign.State{}, err
	}
	if authorize != nil {
		if err := authorize(state); err != nil {
			return "", nil, campaign.State{}, err
		}
	}
	return campaignID, timeline, state, nil
}

func recordsAtOrBeforeSeq(timeline []event.Record, headSeq uint64) []event.Record {
	index := 0
	for index < len(timeline) && timeline[index].Seq <= headSeq {
		index++
	}
	return append([]event.Record(nil), timeline[:index]...)
}

// ListCampaigns returns the most recent persisted campaigns visible to the caller.
func (s *Service) ListCampaigns(ctx context.Context, act caller.Caller) ([]CampaignSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := authz.RequireCaller(act); err != nil {
		return nil, err
	}
	return s.projections.ListCampaignsBySubject(ctx, act.SubjectID, listCampaignsLimit)
}

func normalizeCampaignID(campaignID string) (string, error) {
	if err := canonical.ValidateID(campaignID, "campaign id"); err != nil {
		return "", err
	}
	return campaignID, nil
}

func (s *Service) replayLocked(timeline []event.Record) (campaign.State, error) {
	state := campaign.NewState()
	for _, record := range timeline {
		validated, _, err := s.events.Validate(record.Envelope)
		if err != nil {
			return campaign.State{}, err
		}
		if err := s.registry.Fold(&state, validated); err != nil {
			return campaign.State{}, err
		}
	}
	return state, nil
}
