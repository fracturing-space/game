package service

import (
	"context"
	"fmt"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

type planResult struct {
	plan       PlannedCommand
	finalState campaign.State
	campaignID string
}

func (s *Service) planCurrentLocked(ctx context.Context, act caller.Caller, envelope command.Envelope, ids IDSession) (planResult, error) {
	if err := ctx.Err(); err != nil {
		return planResult{}, err
	}

	validated, _, err := s.commands.Validate(envelope)
	if err != nil {
		return planResult{}, err
	}
	if validated.CampaignID == "" {
		return s.planCurrentValidatedInSlot(ctx, nil, act, validated, ids)
	}

	slot, release := s.acquireCampaignSlot(validated.CampaignID)
	defer release()
	slot.mu.Lock()
	defer slot.mu.Unlock()
	return s.planCurrentValidatedInSlot(ctx, slot, act, validated, ids)
}

func (s *Service) planCurrentValidatedInSlot(ctx context.Context, slot *campaignSlot, act caller.Caller, validated command.Envelope, ids IDSession) (planResult, error) {
	_, ok := s.admission.RuleFor(validated.Type())
	if !ok {
		return planResult{}, fmt.Errorf("admission rule is not registered: %s", validated.Type())
	}

	state := campaign.NewState()
	if validated.CampaignID != "" {
		_, loadedState, err := s.loadCampaignInSlot(ctx, slot, validated.CampaignID)
		if err != nil {
			return planResult{}, err
		}
		state = loadedState
	}

	if _, err := s.admission.Admit(act, state, validated); err != nil {
		return planResult{}, err
	}

	planned, campaignID, finalState, err := s.planValidatedLocked(ctx, act, validated, state, ids)
	if err != nil {
		return planResult{}, err
	}
	return planResult{
		plan:       planned,
		finalState: finalState,
		campaignID: campaignID,
	}, nil
}

func (s *Service) planValidatedLocked(ctx context.Context, act caller.Caller, validated command.Envelope, state campaign.State, ids IDSession) (PlannedCommand, string, campaign.State, error) {
	if err := ctx.Err(); err != nil {
		return PlannedCommand{}, "", campaign.State{}, err
	}

	planned, err := s.registry.Decide(state.Clone(), act, validated, ids.NewID)
	if err != nil {
		return PlannedCommand{}, "", campaign.State{}, err
	}
	if len(planned) == 0 {
		return PlannedCommand{}, "", campaign.State{}, fmt.Errorf("accepted command must emit at least one event")
	}

	campaignID, validatedEvents, err := s.validatePlannedEventsLocked(planned)
	if err != nil {
		return PlannedCommand{}, "", campaign.State{}, err
	}
	finalState := state.Clone()
	for _, plannedEvent := range validatedEvents {
		if err := s.registry.Fold(&finalState, plannedEvent); err != nil {
			return PlannedCommand{}, "", campaign.State{}, err
		}
	}
	return PlannedCommand{
		Accepted: true,
		Events:   validatedEvents,
		State:    campaign.SnapshotOf(finalState),
	}, campaignID, finalState, nil
}

func (s *Service) validatePlannedEventsLocked(planned []event.Envelope) (string, []event.Envelope, error) {
	validated := make([]event.Envelope, 0, len(planned))
	campaignID := ""
	for _, envelope := range planned {
		next, _, err := s.events.Validate(envelope)
		if err != nil {
			return "", nil, err
		}
		if campaignID == "" {
			campaignID = next.CampaignID
		}
		if next.CampaignID != campaignID {
			return "", nil, fmt.Errorf("planned events must target a single campaign")
		}
		validated = append(validated, next)
	}
	return campaignID, validated, nil
}
