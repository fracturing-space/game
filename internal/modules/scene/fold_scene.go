package scene

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
)

func foldCreated(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[scene.Created](envelope)
	if err != nil {
		return err
	}
	state.Scenes[message.SceneID] = scene.Record{
		ID:           message.SceneID,
		SessionID:    message.SessionID,
		Name:         message.Name,
		CharacterIDs: append([]string(nil), message.CharacterIDs...),
	}
	return nil
}

func foldActivated(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[scene.Activated](envelope)
	if err != nil {
		return err
	}
	for id, record := range state.Scenes {
		record.Active = false
		state.Scenes[id] = record
	}
	record, ok := state.Scenes[message.SceneID]
	if !ok {
		return errs.NotFoundf("scene %s not found", message.SceneID)
	}
	record.Active = true
	state.Scenes[message.SceneID] = record
	state.ActiveSceneID = message.SceneID
	return nil
}

func foldEnded(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[scene.Ended](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Scenes[message.SceneID]
	if !ok {
		return errs.NotFoundf("scene %s not found", message.SceneID)
	}
	record.Ended = true
	record.Active = false
	state.Scenes[message.SceneID] = record
	if state.ActiveSceneID == message.SceneID {
		state.ActiveSceneID = ""
	}
	return nil
}

func foldCastReplaced(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[scene.CastReplaced](envelope)
	if err != nil {
		return err
	}
	record, ok := state.Scenes[message.SceneID]
	if !ok {
		return errs.NotFoundf("scene %s not found", message.SceneID)
	}
	record.CharacterIDs = append([]string(nil), message.CharacterIDs...)
	state.Scenes[message.SceneID] = record
	return nil
}
