package gamev1

import (
	"fmt"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/session"
)

func protoAccess(access participant.Access) (gamev1.ParticipantAccess, error) {
	switch access {
	case participant.AccessOwner:
		return gamev1.ParticipantAccess_PARTICIPANT_ACCESS_OWNER, nil
	case participant.AccessMember:
		return gamev1.ParticipantAccess_PARTICIPANT_ACCESS_MEMBER, nil
	default:
		return gamev1.ParticipantAccess_PARTICIPANT_ACCESS_UNSPECIFIED, fmt.Errorf("participant access is invalid: %s", access)
	}
}

func protoCampaignPlayState(playState campaign.PlayState) (gamev1.CampaignPlayState, error) {
	switch playState {
	case campaign.PlayStateSetup:
		return gamev1.CampaignPlayState_CAMPAIGN_PLAY_STATE_SETUP, nil
	case campaign.PlayStateActive:
		return gamev1.CampaignPlayState_CAMPAIGN_PLAY_STATE_ACTIVE, nil
	case campaign.PlayStatePaused:
		return gamev1.CampaignPlayState_CAMPAIGN_PLAY_STATE_PAUSED, nil
	default:
		return gamev1.CampaignPlayState_CAMPAIGN_PLAY_STATE_UNSPECIFIED, fmt.Errorf("campaign play state is invalid: %s", playState)
	}
}

func protoSession(id string, name string, status session.Status, assignments []session.CharacterControllerAssignment) (*gamev1.Session, error) {
	protoStatus, err := protoSessionStatus(status)
	if err != nil {
		return nil, err
	}
	return &gamev1.Session{
		Id:                   id,
		Name:                 name,
		Status:               protoStatus,
		CharacterControllers: protoSessionCharacterControllers(assignments),
	}, nil
}

func protoSessionPayload(id string, name string, assignments []session.CharacterControllerAssignment) *gamev1.SessionStarted {
	return &gamev1.SessionStarted{
		SessionId:            id,
		Name:                 name,
		CharacterControllers: protoSessionCharacterControllers(assignments),
	}
}

func protoSessionCharacterControllers(input []session.CharacterControllerAssignment) []*gamev1.SessionCharacterControllerAssignment {
	if len(input) == 0 {
		return nil
	}
	output := make([]*gamev1.SessionCharacterControllerAssignment, 0, len(input))
	for _, next := range input {
		output = append(output, &gamev1.SessionCharacterControllerAssignment{
			CharacterId:   next.CharacterID,
			ParticipantId: next.ParticipantID,
		})
	}
	return output
}

func protoSessionStatus(statusValue session.Status) (gamev1.SessionStatus, error) {
	switch statusValue {
	case session.StatusActive:
		return gamev1.SessionStatus_SESSION_STATUS_ACTIVE, nil
	case session.StatusEnded:
		return gamev1.SessionStatus_SESSION_STATUS_ENDED, nil
	default:
		return gamev1.SessionStatus_SESSION_STATUS_UNSPECIFIED, fmt.Errorf("session status is invalid: %s", statusValue)
	}
}
