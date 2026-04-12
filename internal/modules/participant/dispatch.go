package participant

import (
	"fmt"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

var participantCommandHandlers = map[command.Type]func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error){
	participant.CommandTypeJoin: decideJoin,
	participant.CommandTypeUpdate: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideUpdate(state, envelope)
	},
	participant.CommandTypeBind: func(state campaign.State, act caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideBind(state, act, envelope)
	},
	participant.CommandTypeUnbind: func(state campaign.State, act caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideUnbind(state, act, envelope)
	},
	participant.CommandTypeLeave: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideLeave(state, envelope)
	},
}

var participantEventHandlers = map[event.Type]func(*campaign.State, event.Envelope) error{
	participant.EventTypeJoined:  foldJoined,
	participant.EventTypeUpdated: foldUpdated,
	participant.EventTypeBound:   foldBound,
	participant.EventTypeUnbound: foldUnbound,
	participant.EventTypeLeft:    foldLeft,
}

// Decide routes the validated participant command into one or more events.
func (Module) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	handler, ok := participantCommandHandlers[envelope.Type()]
	if !ok {
		return nil, fmt.Errorf("participant command is not handled: %s", envelope.Type())
	}
	return handler(state, act, envelope, ids)
}

// Fold applies one participant event in place.
func (Module) Fold(state *campaign.State, envelope event.Envelope) error {
	if state == nil {
		return fmt.Errorf("campaign state is required")
	}
	handler, ok := participantEventHandlers[envelope.Type()]
	if !ok {
		return fmt.Errorf("participant event is not handled: %s", envelope.Type())
	}
	return handler(state, envelope)
}
