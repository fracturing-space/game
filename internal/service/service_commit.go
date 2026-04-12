package service

import (
	"context"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
)

// CommitCommand validates one command and appends its event batch to the live
// journal.
func (s *Service) CommitCommand(ctx context.Context, act caller.Caller, envelope command.Envelope) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	logEnvelope := envelope
	validated, _, err := s.commands.Validate(envelope)
	if err != nil {
		s.logCommandRejected(ctx, act, logEnvelope, err)
		return Result{}, err
	}

	session := s.ids.Session(true)
	if validated.CampaignID == "" {
		return s.executeNewCampaign(ctx, act, logEnvelope, validated, session)
	}

	slot, release := s.acquireCampaignSlot(validated.CampaignID)
	defer release()
	slot.mu.Lock()
	defer slot.mu.Unlock()

	plan, err := s.planCurrentValidatedInSlot(ctx, slot, act, validated, session)
	if err != nil {
		s.logCommandRejected(ctx, act, logEnvelope, err)
		return Result{}, err
	}
	s.logCommandAccepted(ctx, act, logEnvelope, plan.campaignID, plan.plan.Events)

	records, err := s.store.AppendCommits(ctx, plan.campaignID, []PreparedCommit{{Events: plan.plan.Events}}, s.recordClock.Now)
	if err != nil {
		return Result{}, err
	}
	if err := s.persistProjectionInSlot(ctx, slot, plan.campaignID, plan.finalState, headSeqOf(records), lastActivityAtOfTimeline(records)); err != nil {
		return Result{}, err
	}
	session.Commit()
	s.logLiveEventBatch(ctx, plan.campaignID, records)
	return Result{
		Accepted:     true,
		Events:       cloneEnvelopes(plan.plan.Events),
		StoredEvents: cloneEventRecords(records),
		State:        plan.plan.State,
	}, nil
}

func (s *Service) executeNewCampaign(ctx context.Context, act caller.Caller, logEnvelope command.Envelope, validated command.Envelope, session IDSession) (Result, error) {
	plan, err := s.planCurrentValidatedInSlot(ctx, nil, act, validated, session)
	if err != nil {
		s.logCommandRejected(ctx, act, logEnvelope, err)
		return Result{}, err
	}

	slot, release := s.acquireCampaignSlot(plan.campaignID)
	defer release()
	slot.mu.Lock()
	defer slot.mu.Unlock()

	if _, ok, err := s.store.HeadSeq(ctx, plan.campaignID); err != nil {
		s.logCommandRejected(ctx, act, logEnvelope, err)
		return Result{}, err
	} else if ok {
		err := errs.AlreadyExistsf("campaign %s already exists", plan.campaignID)
		s.logCommandRejected(ctx, act, logEnvelope, err)
		return Result{}, err
	}

	s.logCommandAccepted(ctx, act, logEnvelope, plan.campaignID, plan.plan.Events)

	records, err := s.store.AppendCommits(ctx, plan.campaignID, []PreparedCommit{{Events: plan.plan.Events}}, s.recordClock.Now)
	if err != nil {
		return Result{}, err
	}
	if err := s.persistProjectionInSlot(ctx, slot, plan.campaignID, plan.finalState, headSeqOf(records), lastActivityAtOfTimeline(records)); err != nil {
		return Result{}, err
	}
	session.Commit()
	s.logLiveEventBatch(ctx, plan.campaignID, records)
	return Result{
		Accepted:     true,
		Events:       cloneEnvelopes(plan.plan.Events),
		StoredEvents: cloneEventRecords(records),
		State:        plan.plan.State,
	}, nil
}
