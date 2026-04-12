package session

import (
	"fmt"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/session"
)

var sessionCommandHandlers = map[command.Type]func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error){
	session.CommandTypeStart: func(state campaign.State, _ caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
		return decideStart(state, envelope, ids)
	},
	session.CommandTypeEnd: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideEnd(state, envelope)
	},
}

var sessionEventHandlers = map[event.Type]func(*campaign.State, event.Envelope) error{
	session.EventTypeStarted: foldStarted,
	session.EventTypeEnded:   foldEnded,
}

// Decide routes the validated session command into one or more events.
func (Module) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	handler, ok := sessionCommandHandlers[envelope.Type()]
	if !ok {
		return nil, fmt.Errorf("session command is not handled: %s", envelope.Type())
	}
	return handler(state, act, envelope, ids)
}

// Fold applies one session event in place.
func (Module) Fold(state *campaign.State, envelope event.Envelope) error {
	if state == nil {
		return fmt.Errorf("campaign state is required")
	}
	handler, ok := sessionEventHandlers[envelope.Type()]
	if !ok {
		return fmt.Errorf("session event is not handled: %s", envelope.Type())
	}
	return handler(state, envelope)
}
