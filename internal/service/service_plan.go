package service

import (
	"context"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
)

func (s *Service) PlanCommands(ctx context.Context, act caller.Caller, commands []command.Envelope) (CommandPlan, error) {
	if err := ctx.Err(); err != nil {
		return CommandPlan{}, err
	}
	if len(commands) == 0 {
		return CommandPlan{}, errs.InvalidArgumentf("plan commands are required")
	}

	validated := make([]command.Envelope, 0, len(commands))
	campaignID := ""
	for _, envelope := range commands {
		next, _, err := s.commands.Validate(envelope)
		if err != nil {
			return CommandPlan{}, err
		}
		if next.CampaignID == "" {
			return CommandPlan{}, errs.InvalidArgumentf("plan commands must target an existing campaign")
		}
		if campaignID == "" {
			campaignID = next.CampaignID
		} else if next.CampaignID != campaignID {
			return CommandPlan{}, errs.InvalidArgumentf("plan commands must target a single campaign")
		}
		validated = append(validated, next)
	}

	slot, release := s.acquireCampaignSlot(campaignID)
	defer release()
	slot.mu.Lock()
	defer slot.mu.Unlock()

	_, timeline, state, err := s.readAuthorizedCampaignInSlot(ctx, slot, campaignID, func(state campaign.State) error {
		return authz.RequirePlanCommands(act, state)
	})
	if err != nil {
		return CommandPlan{}, err
	}
	if state.PlayState != campaign.PlayStateActive {
		return CommandPlan{}, errs.FailedPreconditionf("command planning is only allowed in %s play state", campaign.PlayStateActive)
	}
	baseSeq := headSeqOf(timeline)
	working := state.Clone()
	session := s.ids.Session(true)
	commits := make([]PreparedCommit, 0, len(validated))
	for _, envelope := range validated {
		rule, ok := s.admission.RuleFor(envelope.Type())
		if !ok {
			return CommandPlan{}, errs.InvalidArgumentf("planning rule is not registered for %s", envelope.Type())
		}
		if !rule.SupportsPlanning {
			return CommandPlan{}, errs.InvalidArgumentf("command %s cannot be planned", envelope.Type())
		}
		if _, err := s.admission.Admit(act, working, envelope); err != nil {
			return CommandPlan{}, err
		}
		planned, _, finalState, err := s.planValidatedLocked(ctx, act, envelope, working, session)
		if err != nil {
			return CommandPlan{}, err
		}
		working = finalState
		commits = append(commits, PreparedCommit{Events: cloneEnvelopes(planned.Events)})
	}

	token, err := session.NewID("plan")
	if err != nil {
		return CommandPlan{}, err
	}
	now := s.recordClock.Now()
	if err := s.storePlan(preparedPlan{
		token:      token,
		caller:     act,
		campaignID: campaignID,
		baseSeq:    baseSeq,
		commits:    clonePreparedCommits(commits),
		state:      working.Clone(),
		expiresAt:  now.Add(s.planTTL),
	}, now); err != nil {
		return CommandPlan{}, err
	}
	session.Commit()
	return CommandPlan{
		Token:      token,
		CampaignID: campaignID,
		BaseSeq:    baseSeq,
		Commits:    clonePreparedCommits(commits),
		State:      campaign.SnapshotOf(working),
	}, nil
}

func (s *Service) ExecutePlan(ctx context.Context, act caller.Caller, token string) (ExecutedPlan, error) {
	if err := ctx.Err(); err != nil {
		return ExecutedPlan{}, err
	}
	if canonical.IsBlank(token) {
		return ExecutedPlan{}, errs.InvalidArgumentf("plan token is required")
	}
	if !canonical.IsExact(token) {
		return ExecutedPlan{}, errs.InvalidArgumentf("plan token must not contain surrounding whitespace")
	}

	plan, err := s.consumePlan(token, act, s.recordClock.Now())
	if err != nil {
		return ExecutedPlan{}, err
	}

	slot, release := s.acquireCampaignSlot(plan.campaignID)
	defer release()
	slot.mu.Lock()
	defer slot.mu.Unlock()

	timeline, _, err := s.loadCampaignInSlot(ctx, slot, plan.campaignID)
	if err != nil {
		return ExecutedPlan{}, err
	}
	if headSeqOf(timeline) != plan.baseSeq {
		return ExecutedPlan{}, errs.Conflictf("campaign %s changed since plan", plan.campaignID)
	}
	records, err := s.store.AppendCommits(ctx, plan.campaignID, clonePreparedCommits(plan.commits), s.recordClock.Now)
	if err != nil {
		return ExecutedPlan{}, err
	}
	if err := s.persistProjectionInSlot(ctx, slot, plan.campaignID, plan.state, headSeqOf(records), lastActivityAtOfTimeline(records)); err != nil {
		return ExecutedPlan{}, err
	}
	return ExecutedPlan{
		CampaignID: plan.campaignID,
		HeadSeq:    headSeqOf(records),
		State:      campaign.SnapshotOf(plan.state),
	}, nil
}
