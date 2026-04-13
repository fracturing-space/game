package util

import (
	"fmt"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

// DefaultAssignments derives the active session controller defaults from the
// current campaign state.
func DefaultAssignments(state campaign.State) ([]session.CharacterControllerAssignment, error) {
	assignments := make(map[string]string, len(state.Characters))
	for _, record := range state.Characters {
		if !record.Active {
			continue
		}
		participantID := record.ParticipantID
		if participantID == "" {
			return nil, errs.FailedPreconditionf("session character controller defaults are incomplete")
		}
		participantRecord, ok := state.Participants[participantID]
		if !ok || !participantRecord.Active {
			return nil, errs.FailedPreconditionf("session character controller references inactive participant %s", participantID)
		}
		assignments[record.ID] = participantID
	}
	result := make([]session.CharacterControllerAssignment, 0, len(assignments))
	for characterID, participantID := range assignments {
		if characterID == "" || participantID == "" {
			return nil, errs.FailedPreconditionf("session character controller defaults are incomplete")
		}
		result = append(result, session.CharacterControllerAssignment{
			CharacterID:   characterID,
			ParticipantID: participantID,
		})
	}
	return session.CloneAssignments(result), nil
}

// EffectiveAssignments overlays explicit session controller overrides on the
// campaign-derived defaults.
func EffectiveAssignments(state campaign.State, overrides []session.CharacterControllerAssignment) ([]session.CharacterControllerAssignment, error) {
	defaults, err := DefaultAssignments(state)
	if err != nil {
		return nil, err
	}
	assignments := make(map[string]string, len(defaults))
	for _, assignment := range defaults {
		assignments[assignment.CharacterID] = assignment.ParticipantID
	}
	for _, override := range overrides {
		characterID := override.CharacterID
		record, ok := state.Characters[characterID]
		if !ok || !record.Active {
			return nil, errs.InvalidArgumentf("session character controller references unknown character %s", characterID)
		}
		participantID := override.ParticipantID
		participantRecord, ok := state.Participants[participantID]
		if !ok || !participantRecord.Active {
			return nil, errs.InvalidArgumentf("session character controller references unknown participant %s", participantID)
		}
		assignments[characterID] = participantID
	}
	result := make([]session.CharacterControllerAssignment, 0, len(assignments))
	for characterID, participantID := range assignments {
		if characterID == "" || participantID == "" {
			return nil, errs.FailedPreconditionf("session character controller defaults are incomplete")
		}
		result = append(result, session.CharacterControllerAssignment{
			CharacterID:   characterID,
			ParticipantID: participantID,
		})
	}
	return session.CloneAssignments(result), nil
}

// BuildPlayStartEvent emits the session.started event and folded record used to
// enter play.
func BuildPlayStartEvent(state campaign.State, campaignID, sessionID, name string, overrides []session.CharacterControllerAssignment) (event.Envelope, session.Record, error) {
	assignments, err := EffectiveAssignments(state, overrides)
	if err != nil {
		return event.Envelope{}, session.Record{}, err
	}
	if name == "" {
		name = fmt.Sprintf("Session %d", state.SessionCount+1)
	}
	started, err := event.NewEnvelope(
		session.StartedEventSpec,
		campaignID,
		session.Started{
			SessionID:            sessionID,
			Name:                 name,
			CharacterControllers: assignments,
		},
	)
	if err != nil {
		return event.Envelope{}, session.Record{}, err
	}
	return started, session.Record{
		ID:                   sessionID,
		Name:                 name,
		Status:               session.StatusActive,
		CharacterControllers: session.CloneAssignments(assignments),
	}, nil
}

// BuildPlayEndEvents emits the scene/session/campaign events needed to end the
// active play session and return to setup.
func BuildPlayEndEvents(state campaign.State, campaignID string) ([]event.Envelope, error) {
	activeSession := state.ActiveSession()
	if activeSession == nil {
		return nil, errs.NotFoundf("campaign %s active session not found", state.CampaignID)
	}

	events := make([]event.Envelope, 0, 3)
	activeSceneID := state.ActiveSceneID
	if activeSceneID != "" {
		if _, ok := state.Scenes[activeSceneID]; ok {
			sceneEnded, err := event.NewEnvelope(
				scene.EndedEventSpec,
				campaignID,
				scene.Ended{SceneID: activeSceneID},
			)
			if err != nil {
				return nil, err
			}
			events = append(events, sceneEnded)
		}
	}
	ended, err := event.NewEnvelope(
		session.EndedEventSpec,
		campaignID,
		session.Ended{
			SessionID:            activeSession.ID,
			Name:                 activeSession.Name,
			CharacterControllers: session.CloneAssignments(activeSession.CharacterControllers),
		},
	)
	if err != nil {
		return nil, err
	}
	events = append(events, ended)
	playEnded, err := event.NewEnvelope(
		campaign.PlayEndedEventSpec,
		campaignID,
		campaign.PlayEnded{
			SessionID: activeSession.ID,
			SceneID:   activeSceneID,
		},
	)
	if err != nil {
		return nil, err
	}
	return append(events, playEnded), nil
}
