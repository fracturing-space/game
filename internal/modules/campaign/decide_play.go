package campaign

import (
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/modules/util"
	"github.com/fracturing-space/game/internal/readiness"
	"github.com/fracturing-space/game/internal/session"
)

func decidePlayBegin(state campaign.State, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	activeSession := state.ActiveSession()
	events := make([]event.Envelope, 0, 3)
	if activeSession == nil {
		sessionEvents, sessionRecord, err := newSessionStartedEvents(state, envelope, ids)
		if err != nil {
			return nil, err
		}
		activeSession = sessionRecord
		events = append(events, sessionEvents...)
	}
	if rejection := readiness.EvaluatePlayTransition(state); rejection != nil {
		return nil, rejection
	}
	began, err := event.NewEnvelope(
		campaign.PlayBeganEventSpec,
		envelope.CampaignID,
		campaign.PlayBegan{
			SessionID: activeSession.ID,
			SceneID:   state.ActiveSceneID,
		},
	)
	if err != nil {
		return nil, err
	}
	return append(events, began), nil
}

func decidePlayEnd(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	return util.BuildPlayEndEvents(state, envelope.CampaignID)
}

func decidePlayPause(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[campaign.PlayPause](envelope)
	if err != nil {
		return nil, err
	}
	activeSession := state.ActiveSession()
	if activeSession == nil {
		return nil, errs.FailedPreconditionf("active session is required to pause play")
	}
	if state.ActiveSceneID == "" {
		return nil, errs.FailedPreconditionf("active scene is required to pause play")
	}
	paused, err := event.NewEnvelope(
		campaign.PlayPausedEventSpec,
		envelope.CampaignID,
		campaign.PlayPaused{
			SessionID: activeSession.ID,
			SceneID:   state.ActiveSceneID,
			Reason:    message.Reason,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{paused}, nil
}

func decidePlayResume(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[campaign.PlayResume](envelope)
	if err != nil {
		return nil, err
	}
	activeSession := state.ActiveSession()
	if activeSession == nil {
		return nil, errs.FailedPreconditionf("active session is required to resume play")
	}
	if state.ActiveSceneID == "" {
		return nil, errs.FailedPreconditionf("active scene is required to resume play")
	}
	resumed, err := event.NewEnvelope(
		campaign.PlayResumedEventSpec,
		envelope.CampaignID,
		campaign.PlayResumed{
			SessionID: activeSession.ID,
			SceneID:   state.ActiveSceneID,
			Reason:    message.Reason,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{resumed}, nil
}

func newSessionStartedEvents(state campaign.State, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, *session.Record, error) {
	sessionID, err := ids("sess")
	if err != nil {
		return nil, nil, err
	}
	started, record, err := util.BuildPlayStartEvent(state, envelope.CampaignID, sessionID, "", nil)
	if err != nil {
		return nil, nil, err
	}
	return []event.Envelope{started}, &record, nil
}
