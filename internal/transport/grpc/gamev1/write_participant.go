package gamev1

import (
	"context"
	"fmt"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/participant"
)

// CreateParticipant creates one campaign participant.
func (s *Server) CreateParticipant(ctx context.Context, req *gamev1.CreateParticipantRequest) (*gamev1.CreateParticipantResponse, error) {
	result, err := executeCommand(s, ctx, req, createParticipantEnvelope)
	if err != nil {
		return nil, err
	}
	participantID, err := resultParticipantID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.CreateParticipantResponse{ParticipantId: participantID}, nil
}

// UpdateParticipant updates one campaign participant.
func (s *Server) UpdateParticipant(ctx context.Context, req *gamev1.UpdateParticipantRequest) (*gamev1.UpdateParticipantResponse, error) {
	_, err := executeCommand(s, ctx, req, updateParticipantEnvelope)
	if err != nil {
		return nil, err
	}
	return &gamev1.UpdateParticipantResponse{ParticipantId: req.GetParticipantId()}, nil
}

// BindParticipant binds the caller to one human participant.
func (s *Server) BindParticipant(ctx context.Context, req *gamev1.BindParticipantRequest) (*gamev1.BindParticipantResponse, error) {
	svc, err := s.requireService()
	if err != nil {
		return nil, err
	}
	if err := requireRequest(req); err != nil {
		return nil, err
	}
	envelope, err := bindParticipantEnvelope(req)
	if err != nil {
		return nil, invalidArgument(err)
	}
	act, err := callerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	message, ok := envelope.Message.(participant.Bind)
	if !ok {
		return nil, internalStatus(fmt.Errorf("participant bind command is required"))
	}
	message.SubjectID = act.SubjectID
	envelope.Message = message
	result, err := svc.CommitCommand(ctx, act, envelope)
	if err != nil {
		return nil, mapDomainError(err)
	}
	participantID, err := resultParticipantEventID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.BindParticipantResponse{ParticipantId: participantID}, nil
}

// UnbindParticipant clears one human participant binding.
func (s *Server) UnbindParticipant(ctx context.Context, req *gamev1.UnbindParticipantRequest) (*gamev1.UnbindParticipantResponse, error) {
	result, err := executeCommand(s, ctx, req, unbindParticipantEnvelope)
	if err != nil {
		return nil, err
	}
	participantID, err := resultParticipantEventID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.UnbindParticipantResponse{ParticipantId: participantID}, nil
}

// DeleteParticipant removes one participant.
func (s *Server) DeleteParticipant(ctx context.Context, req *gamev1.DeleteParticipantRequest) (*gamev1.DeleteParticipantResponse, error) {
	result, err := executeCommand(s, ctx, req, deleteParticipantEnvelope)
	if err != nil {
		return nil, err
	}
	participantID, err := resultParticipantEventID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.DeleteParticipantResponse{ParticipantId: participantID}, nil
}

func createParticipantEnvelope(req *gamev1.CreateParticipantRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	access, err := domainAccess(req.GetAccess())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: participant.Join{
			Name:   req.GetName(),
			Access: access,
		},
	}, nil
}

func updateParticipantEnvelope(req *gamev1.UpdateParticipantRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	access, err := domainAccess(req.GetAccess())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: participant.Update{
			ParticipantID: req.GetParticipantId(),
			Name:          req.GetName(),
			Access:        access,
		},
	}, nil
}

func bindParticipantEnvelope(req *gamev1.BindParticipantRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: participant.Bind{
			ParticipantID: req.GetParticipantId(),
		},
	}, nil
}

func unbindParticipantEnvelope(req *gamev1.UnbindParticipantRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: participant.Unbind{
			ParticipantID: req.GetParticipantId(),
		},
	}, nil
}

func deleteParticipantEnvelope(req *gamev1.DeleteParticipantRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: participant.Leave{
			ParticipantID: req.GetParticipantId(),
			Reason:        req.GetReason(),
		},
	}, nil
}
