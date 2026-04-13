package scene

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/scene"
)

func requireSceneInActiveSession(state campaign.State, envelope command.Envelope) (scene.Record, error) {
	activeSession := state.ActiveSession()
	if activeSession == nil {
		return scene.Record{}, errs.FailedPreconditionf("active session is required")
	}
	switch message := envelope.Message.(type) {
	case scene.Activate:
		return validateSceneLookup(state, message.SceneID)
	case scene.End:
		return validateSceneLookup(state, message.SceneID)
	case scene.ReplaceCast:
		return validateSceneLookup(state, message.SceneID)
	default:
		return scene.Record{}, errs.InvalidArgumentf("scene id is required")
	}
}

func validateSceneLookup(state campaign.State, sceneID string) (scene.Record, error) {
	record, ok := state.Scenes[sceneID]
	if !ok {
		return scene.Record{}, errs.NotFoundf("scene %s not found", sceneID)
	}
	activeSession := state.ActiveSession()
	if activeSession == nil || record.SessionID != activeSession.ID {
		return scene.Record{}, errs.NotFoundf("scene %s not found", sceneID)
	}
	return record, nil
}

func validateCast(state campaign.State, characterIDs []string) error {
	for _, characterID := range characterIDs {
		record, ok := state.Characters[characterID]
		if !ok || !record.Active {
			return errs.InvalidArgumentf("scene cast references unknown character %s", characterID)
		}
	}
	return nil
}
