package gamev1

import (
	"context"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
)

// CreateCampaign creates a new campaign timeline.
func (s *Server) CreateCampaign(ctx context.Context, req *gamev1.CreateCampaignRequest) (*gamev1.CreateCampaignResponse, error) {
	svc, err := s.requireService()
	if err != nil {
		return nil, err
	}
	if err := requireRequest(req); err != nil {
		return nil, err
	}
	envelope := createCampaignEnvelope(req)
	act, err := callerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	result, err := svc.CommitCommand(ctx, act, envelope)
	if err != nil {
		return nil, mapDomainError(err)
	}
	campaignID, err := resultCampaignID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.CreateCampaignResponse{CampaignId: campaignID}, nil
}

// UpdateCampaign updates campaign metadata.
func (s *Server) UpdateCampaign(ctx context.Context, req *gamev1.UpdateCampaignRequest) (*gamev1.UpdateCampaignResponse, error) {
	result, err := executeCommand(s, ctx, req, updateCampaignEnvelope)
	if err != nil {
		return nil, err
	}
	campaignID, err := resultCampaignID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.UpdateCampaignResponse{CampaignId: campaignID}, nil
}

// BeginPlay transitions the campaign to PLAY.
func (s *Server) BeginPlay(ctx context.Context, req *gamev1.BeginPlayRequest) (*gamev1.BeginPlayResponse, error) {
	result, err := executeCommand(s, ctx, req, beginPlayEnvelope)
	if err != nil {
		return nil, err
	}
	campaignID, err := resultCampaignID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.BeginPlayResponse{CampaignId: campaignID}, nil
}

// EndPlay leaves active play and returns the campaign to setup.
func (s *Server) EndPlay(ctx context.Context, req *gamev1.EndPlayRequest) (*gamev1.EndPlayResponse, error) {
	result, err := executeCommand(s, ctx, req, endPlayEnvelope)
	if err != nil {
		return nil, err
	}
	campaignID, err := resultCampaignID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.EndPlayResponse{CampaignId: campaignID}, nil
}

// SetCampaignAIBinding updates the campaign-level AI agent binding.
func (s *Server) SetCampaignAIBinding(ctx context.Context, req *gamev1.SetCampaignAIBindingRequest) (*gamev1.SetCampaignAIBindingResponse, error) {
	result, err := executeCommand(s, ctx, req, setCampaignAIBindingEnvelope)
	if err != nil {
		return nil, err
	}
	campaignID, err := resultCampaignID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.SetCampaignAIBindingResponse{
		CampaignId: campaignID,
		AiAgentId:  result.State.AIAgentID,
	}, nil
}

// ClearCampaignAIBinding clears the campaign-level AI agent binding.
func (s *Server) ClearCampaignAIBinding(ctx context.Context, req *gamev1.ClearCampaignAIBindingRequest) (*gamev1.ClearCampaignAIBindingResponse, error) {
	result, err := executeCommand(s, ctx, req, clearCampaignAIBindingEnvelope)
	if err != nil {
		return nil, err
	}
	campaignID, err := resultCampaignID(result)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.ClearCampaignAIBindingResponse{CampaignId: campaignID}, nil
}

func createCampaignEnvelope(req *gamev1.CreateCampaignRequest) command.Envelope {
	return command.Envelope{
		Message: campaign.Create{
			Name:      req.GetName(),
			OwnerName: req.GetOwnerName(),
		},
	}
}

func updateCampaignEnvelope(req *gamev1.UpdateCampaignRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message: campaign.Update{
			Name: req.GetName(),
		},
	}, nil
}

func beginPlayEnvelope(req *gamev1.BeginPlayRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message:    campaign.PlayBegin{},
	}, nil
}

func endPlayEnvelope(req *gamev1.EndPlayRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message:    campaign.PlayEnd{},
	}, nil
}

func setCampaignAIBindingEnvelope(req *gamev1.SetCampaignAIBindingRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message:    campaign.AIBind{AIAgentID: req.GetAiAgentId()},
	}, nil
}

func clearCampaignAIBindingEnvelope(req *gamev1.ClearCampaignAIBindingRequest) (command.Envelope, error) {
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return command.Envelope{}, err
	}
	return command.Envelope{
		CampaignID: campaignID,
		Message:    campaign.AIUnbind{},
	}, nil
}
