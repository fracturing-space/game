package service

import (
	"strings"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

type campaignSummaryResource struct {
	Campaign campaignSummaryPayload `json:"campaign"`
}

type campaignSummaryPayload struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	PlayState        campaign.PlayState `json:"play_state"`
	AIAgentID        string             `json:"ai_agent_id"`
	CharacterCount   int                `json:"character_count"`
	ParticipantCount int                `json:"participant_count"`
}

type participantsResource struct {
	Participants []participant.Record `json:"participants"`
}

type charactersResource struct {
	Characters []character.Record `json:"characters"`
}

type characterSheetResource struct {
	Character character.Record `json:"character"`
}

type sessionsResource struct {
	Sessions []session.Record `json:"sessions"`
}

type scenesResource struct {
	Scenes []scene.Record `json:"scenes"`
}

func (s *Service) readCampaignResource(request campaignResourceRequest) (any, error) {
	return campaignSummaryResource{
		Campaign: campaignSummaryPayload{
			ID:               request.snapshot.ID,
			Name:             request.snapshot.Name,
			PlayState:        request.snapshot.PlayState,
			AIAgentID:        request.snapshot.AIAgentID,
			CharacterCount:   len(request.snapshot.Characters),
			ParticipantCount: len(request.snapshot.Participants),
		},
	}, nil
}

func (s *Service) readParticipantsResource(request campaignResourceRequest) (any, error) {
	return participantsResource{Participants: request.snapshot.Participants}, nil
}

func (s *Service) readCharactersResource(request campaignResourceRequest) (any, error) {
	if len(request.parts) == 1 {
		return charactersResource{Characters: request.snapshot.Characters}, nil
	}
	if len(request.parts) != 3 || request.parts[2] != "sheet" {
		return nil, errs.InvalidArgumentf("unknown character resource uri: %s", request.uri)
	}

	characterID := request.parts[1]
	charRecord, ok := request.state.Characters[characterID]
	if !ok {
		return nil, errs.NotFoundf("character %s not found", characterID)
	}
	return characterSheetResource{Character: charRecord}, nil
}

func (s *Service) readSessionsResource(request campaignResourceRequest) (any, error) {
	if len(request.parts) == 1 {
		return sessionsResource{Sessions: append([]session.Record(nil), request.snapshot.Sessions...)}, nil
	}
	if len(request.parts) == 3 && request.parts[2] == "scenes" {
		sessionID := request.parts[1]
		if request.snapshot.Session(sessionID) == nil {
			return nil, errs.NotFoundf("session %s not found", sessionID)
		}
		return scenesResource{Scenes: request.snapshot.ScenesForSession(sessionID)}, nil
	}
	return nil, errs.InvalidArgumentf("unknown session resource uri: %s", request.uri)
}

func (s *Service) readArtifactsResource(request campaignResourceRequest) (any, error) {
	if len(request.parts) < 2 {
		return nil, errs.InvalidArgumentf("artifact path is required")
	}

	path := strings.Join(request.parts[1:], "/")
	item, ok, err := s.artifacts.GetArtifact(request.ctx, request.campaignID, path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.NotFoundf("artifact %s not found", path)
	}
	return rawResource(item.Content), nil
}
