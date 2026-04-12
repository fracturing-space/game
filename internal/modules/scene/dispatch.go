package scene

import (
	"fmt"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
)

var sceneCommandHandlers = map[command.Type]func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error){
	scene.CommandTypeCreate: func(state campaign.State, _ caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
		return decideCreate(state, envelope, ids)
	},
	scene.CommandTypeActivate: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideActivate(state, envelope)
	},
	scene.CommandTypeEnd: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideEnd(state, envelope)
	},
	scene.CommandTypeReplaceCast: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideReplaceCast(state, envelope)
	},
}

var sceneEventHandlers = map[event.Type]func(*campaign.State, event.Envelope) error{
	scene.EventTypeCreated:      foldCreated,
	scene.EventTypeActivated:    foldActivated,
	scene.EventTypeEnded:        foldEnded,
	scene.EventTypeCastReplaced: foldCastReplaced,
}

// Decide routes the validated scene command into one or more events.
func (Module) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	handler, ok := sceneCommandHandlers[envelope.Type()]
	if !ok {
		return nil, fmt.Errorf("scene command is not handled: %s", envelope.Type())
	}
	return handler(state, act, envelope, ids)
}

// Fold applies one scene event in place.
func (Module) Fold(state *campaign.State, envelope event.Envelope) error {
	if state == nil {
		return fmt.Errorf("campaign state is required")
	}
	handler, ok := sceneEventHandlers[envelope.Type()]
	if !ok {
		return fmt.Errorf("scene event is not handled: %s", envelope.Type())
	}
	return handler(state, envelope)
}
