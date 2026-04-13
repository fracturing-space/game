package campaign

import (
	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
)

var _ engine.Module = Module{}

// Module owns campaign-specific command and event behavior.
type Module struct{}

// New returns the campaign module.
func New() Module {
	return Module{}
}

// Name returns the module label.
func (Module) Name() string { return "core.campaign" }

// Commands returns the registered command specs.
func (Module) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{
		{
			Spec: campaign.CreateCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireCreateCampaign(act)
				},
			},
		},
		{
			Spec: campaign.UpdateCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireUpdateCampaign(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: campaign.AIBindCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireManageAIBinding(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: campaign.AIUnbindCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireManageAIBinding(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: campaign.PlayBeginCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireBeginPlay(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: campaign.PlayEndCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireEndPlay(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive, campaign.PlayStatePaused},
			},
		},
		{
			Spec: campaign.PlayPauseCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequirePausePlay(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive},
			},
		},
		{
			Spec: campaign.PlayResumeCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireResumePlay(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStatePaused},
			},
		},
	}
}

// Events returns the registered event specs.
func (Module) Events() []event.Spec {
	return []event.Spec{
		campaign.CreatedEventSpec,
		campaign.UpdatedEventSpec,
		campaign.AIBoundEventSpec,
		campaign.AIUnboundEventSpec,
		campaign.PlayBeganEventSpec,
		campaign.PlayPausedEventSpec,
		campaign.PlayResumedEventSpec,
		campaign.PlayEndedEventSpec,
	}
}
