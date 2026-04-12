package campaign

import (
	"fmt"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

var campaignCommandHandlers = map[command.Type]func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error){
	campaign.CommandTypeCreate: decideCreate,
	campaign.CommandTypeUpdate: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideUpdate(state, envelope)
	},
	campaign.CommandTypeAIBind: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideAIBind(state, envelope)
	},
	campaign.CommandTypeAIUnbind: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decideAIUnbind(state, envelope)
	},
	campaign.CommandTypePlayBegin: func(state campaign.State, _ caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
		return decidePlayBegin(state, envelope, ids)
	},
	campaign.CommandTypePlayEnd: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decidePlayEnd(state, envelope)
	},
	campaign.CommandTypePlayPause: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decidePlayPause(state, envelope)
	},
	campaign.CommandTypePlayResume: func(state campaign.State, _ caller.Caller, envelope command.Envelope, _ func(string) (string, error)) ([]event.Envelope, error) {
		return decidePlayResume(state, envelope)
	},
}

var campaignEventHandlers = map[event.Type]func(*campaign.State, event.Envelope) error{
	campaign.EventTypeCreated:     foldCreated,
	campaign.EventTypeUpdated:     foldUpdated,
	campaign.EventTypeAIBound:     foldAIBound,
	campaign.EventTypeAIUnbound:   foldAIUnbound,
	campaign.EventTypePlayBegan:   foldPlayBegan,
	campaign.EventTypePlayPaused:  foldPlayPaused,
	campaign.EventTypePlayResumed: foldPlayResumed,
	campaign.EventTypePlayEnded:   foldPlayEnded,
}

// Decide routes the validated campaign command into one or more events.
func (Module) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	handler, ok := campaignCommandHandlers[envelope.Type()]
	if !ok {
		return nil, fmt.Errorf("campaign command is not handled: %s", envelope.Type())
	}
	return handler(state, act, envelope, ids)
}

// Fold applies one campaign event in place.
func (Module) Fold(state *campaign.State, envelope event.Envelope) error {
	if state == nil {
		return fmt.Errorf("campaign state is required")
	}
	handler, ok := campaignEventHandlers[envelope.Type()]
	if !ok {
		return fmt.Errorf("campaign event is not handled: %s", envelope.Type())
	}
	return handler(state, envelope)
}
