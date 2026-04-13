package scene

import (
	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
)

var _ engine.Module = Module{}

// Module owns scene-specific command and event behavior.
type Module struct{}

// New returns the scene module.
func New() Module {
	return Module{}
}

// Name returns the module label.
func (Module) Name() string { return "core.scene" }

// Commands returns the registered command specs.
func (Module) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{
		{
			Spec: scene.CreateCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireCreateScene(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive},
				SupportsPlanning:  true,
			},
		},
		{
			Spec: scene.ActivateCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireActivateScene(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive},
				SupportsPlanning:  true,
			},
		},
		{
			Spec: scene.EndCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireEndScene(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive},
				SupportsPlanning:  true,
			},
		},
		{
			Spec: scene.ReplaceCastCommandSpec,
			Admission: admission.Rule{
				Authorize: func(act caller.Caller, state campaign.State) error {
					return authz.RequireReplaceSceneCast(act, state)
				},
				AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive},
				SupportsPlanning:  true,
			},
		},
	}
}

// Events returns the registered event specs.
func (Module) Events() []event.Spec {
	return []event.Spec{
		scene.CreatedEventSpec,
		scene.ActivatedEventSpec,
		scene.EndedEventSpec,
		scene.CastReplacedEventSpec,
	}
}
