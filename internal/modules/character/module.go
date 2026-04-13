package character

import (
	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
)

var _ engine.Module = Module{}

// Module owns character-specific command and event behavior.
type Module struct{}

// New returns the character module.
func New() Module {
	return Module{}
}

// Name returns the module label.
func (Module) Name() string { return "core.character" }

// Commands returns the registered command specs.
func (Module) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{
		{
			Spec: character.CreateCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireCreateCharacter(act)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: character.UpdateCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireUpdateCharacter(act)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
		{
			Spec: character.DeleteCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireDeleteCharacter(act)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
			},
		},
	}
}

// Events returns the registered event specs.
func (Module) Events() []event.Spec {
	return []event.Spec{
		character.CreatedEventSpec,
		character.UpdatedEventSpec,
		character.DeletedEventSpec,
	}
}
