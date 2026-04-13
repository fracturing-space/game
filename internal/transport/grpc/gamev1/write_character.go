package gamev1

import (
	"context"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
)

// CreateCharacter creates one character.
func (s *Server) CreateCharacter(ctx context.Context, req *gamev1.CreateCharacterRequest) (*gamev1.CreateCharacterResponse, error) {
	result, err := executeCommand(s, ctx, req, createCharacterEnvelope)
	if err != nil {
		return nil, err
	}
	characterID, err := resultCharacterID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.CreateCharacterResponse{CharacterId: characterID}, nil
}

// UpdateCharacter updates one character.
func (s *Server) UpdateCharacter(ctx context.Context, req *gamev1.UpdateCharacterRequest) (*gamev1.UpdateCharacterResponse, error) {
	_, err := executeCommand(s, ctx, req, updateCharacterEnvelope)
	if err != nil {
		return nil, err
	}
	return &gamev1.UpdateCharacterResponse{CharacterId: req.GetCharacterId()}, nil
}

// DeleteCharacter deletes one character.
func (s *Server) DeleteCharacter(ctx context.Context, req *gamev1.DeleteCharacterRequest) (*gamev1.DeleteCharacterResponse, error) {
	result, err := executeCommand(s, ctx, req, deleteCharacterEnvelope)
	if err != nil {
		return nil, err
	}
	characterID, err := resultCharacterEventID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.DeleteCharacterResponse{CharacterId: characterID}, nil
}

func createCharacterEnvelope(req *gamev1.CreateCharacterRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: character.Create{
			ParticipantID: req.GetParticipantId(),
			Name:          req.GetName(),
		},
	}, nil
}

func updateCharacterEnvelope(req *gamev1.UpdateCharacterRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: character.Update{
			CharacterID:   req.GetCharacterId(),
			ParticipantID: req.GetParticipantId(),
			Name:          req.GetName(),
		},
	}, nil
}

func deleteCharacterEnvelope(req *gamev1.DeleteCharacterRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: character.Delete{
			CharacterID: req.GetCharacterId(),
			Reason:      req.GetReason(),
		},
	}, nil
}
