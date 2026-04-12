package session

import (
	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/session"
)

var _ engine.Module = Module{}

// Module owns session lifecycle command and event behavior.
type Module struct{}

// New returns the session module.
func New() Module {
	return Module{}
}

// Name returns the module label.
func (Module) Name() string { return "core.session" }

// Commands returns the registered command specs.
func (Module) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{
		{
			Spec: session.StartCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireStartSession(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: session.EndCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireEndSession(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive, campaign.PlayStatePaused},
			},
		},
	}
}

// Events returns the registered event specs.
func (Module) Events() []event.Spec {
	return []event.Spec{
		session.StartedEventSpec,
		session.EndedEventSpec,
	}
}
