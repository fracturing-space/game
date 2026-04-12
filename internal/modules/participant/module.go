package participant

import (
	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

var _ engine.Module = Module{}

// Module owns participant-specific command and event behavior.
type Module struct{}

// New returns the participant module.
func New() Module {
	return Module{}
}

// Name returns the module label.
func (Module) Name() string { return "core.participant" }

// Commands returns the registered command specs.
func (Module) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{
		{
			Spec: participant.JoinCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireCreateParticipant(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: participant.UpdateCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireUpdateParticipant(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: participant.BindCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireBindParticipant(act)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: participant.UnbindCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireUnbindParticipant(act)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: participant.LeaveCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireDeleteParticipant(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
	}
}

// Events returns the registered event specs.
func (Module) Events() []event.Spec {
	return []event.Spec{
		participant.JoinedEventSpec,
		participant.UpdatedEventSpec,
		participant.BoundEventSpec,
		participant.UnboundEventSpec,
		participant.LeftEventSpec,
	}
}
