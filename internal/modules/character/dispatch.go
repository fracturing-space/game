package character

import (
	"fmt"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

var characterCommandHandlers = map[command.Type]func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error){
	character.CommandTypeCreate: decideCreate,
	character.CommandTypeUpdate: func(state campaign.State, act caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideUpdate(state, act, envelope)
	},
	character.CommandTypeDelete: func(state campaign.State, act caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideDelete(state, act, envelope)
	},
}

var characterEventHandlers = map[event.Type]func(*campaign.State, event.Envelope) error{
	character.EventTypeCreated: foldCreated,
	character.EventTypeUpdated: foldUpdated,
	character.EventTypeDeleted: foldDeleted,
}

// Decide routes the validated character command into one or more events.
func (Module) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	handler, ok := characterCommandHandlers[envelope.Type()]
	if !ok {
		return nil, fmt.Errorf("character command is not handled: %s", envelope.Type())
	}
	return handler(state, act, envelope, ids)
}

// Fold applies one character event in place.
func (Module) Fold(state *campaign.State, envelope event.Envelope) error {
	if state == nil {
		return fmt.Errorf("campaign state is required")
	}
	handler, ok := characterEventHandlers[envelope.Type()]
	if !ok {
		return fmt.Errorf("character event is not handled: %s", envelope.Type())
	}
	return handler(state, envelope)
}
