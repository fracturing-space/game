package scene

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
)

func decideCreate(state campaign.State, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	activeSession := state.ActiveSession()
	if activeSession == nil {
		return nil, errs.FailedPreconditionf("active session is required")
	}
	message, err := command.MessageAs[scene.Create](envelope)
	if err != nil {
		return nil, err
	}
	if err := validateCast(state, message.CharacterIDs); err != nil {
		return nil, err
	}
	sceneID, err := ids("scene")
	if err != nil {
		return nil, err
	}
	created, err := event.NewEnvelope(
		scene.CreatedEventSpec,
		envelope.CampaignID,
		scene.Created{
			SceneID:      sceneID,
			SessionID:    activeSession.ID,
			Name:         message.Name,
			CharacterIDs: message.CharacterIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{created}, nil
}

func decideActivate(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	record, err := requireSceneInActiveSession(state, envelope)
	if err != nil {
		return nil, err
	}
	if record.Ended {
		return nil, errs.FailedPreconditionf("scene %s is already ended", record.ID)
	}
	activated, err := event.NewEnvelope(
		scene.ActivatedEventSpec,
		envelope.CampaignID,
		scene.Activated{SceneID: record.ID},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{activated}, nil
}

func decideEnd(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	record, err := requireSceneInActiveSession(state, envelope)
	if err != nil {
		return nil, err
	}
	if record.Ended {
		return nil, errs.FailedPreconditionf("scene %s is already ended", record.ID)
	}
	ended, err := event.NewEnvelope(
		scene.EndedEventSpec,
		envelope.CampaignID,
		scene.Ended{SceneID: record.ID},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{ended}, nil
}

func decideReplaceCast(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	record, err := requireSceneInActiveSession(state, envelope)
	if err != nil {
		return nil, err
	}
	if record.Ended {
		return nil, errs.FailedPreconditionf("scene %s is already ended", record.ID)
	}
	message, err := command.MessageAs[scene.ReplaceCast](envelope)
	if err != nil {
		return nil, err
	}
	if err := validateCast(state, message.CharacterIDs); err != nil {
		return nil, err
	}
	replaced, err := event.NewEnvelope(
		scene.CastReplacedEventSpec,
		envelope.CampaignID,
		scene.CastReplaced{
			SceneID:      record.ID,
			CharacterIDs: message.CharacterIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{replaced}, nil
}
