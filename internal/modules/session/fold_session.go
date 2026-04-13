package session

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/session"
)

func foldStarted(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[session.Started](envelope)
	if err != nil {
		return err
	}
	state.Sessions[message.SessionID] = session.Record{
		ID:                   message.SessionID,
		Name:                 message.Name,
		Status:               session.StatusActive,
		CharacterControllers: session.CloneAssignments(message.CharacterControllers),
	}
	state.ActiveSessionID = message.SessionID
	state.ActiveSceneID = ""
	state.SessionCount++
	return nil
}

func foldEnded(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[session.Ended](envelope)
	if err != nil {
		return err
	}
	state.Sessions[message.SessionID] = session.Record{
		ID:                   message.SessionID,
		Name:                 message.Name,
		Status:               session.StatusEnded,
		CharacterControllers: session.CloneAssignments(message.CharacterControllers),
	}
	state.ActiveSessionID = ""
	state.ActiveSceneID = ""
	return nil
}
