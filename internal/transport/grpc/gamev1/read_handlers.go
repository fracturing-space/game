package gamev1

import (
	"context"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
)

// GetCampaign returns the persisted campaign state.
func (s *Server) GetCampaign(ctx context.Context, req *gamev1.GetCampaignRequest) (*gamev1.GetCampaignResponse, error) {
	svc, err := s.requireService()
	if err != nil {
		return nil, err
	}
	if err := requireRequest(req); err != nil {
		return nil, err
	}
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return nil, invalidArgument(err)
	}
	act, err := callerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	inspection, err := svc.Inspect(ctx, act, campaignID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	state, err := protoState(inspection.State)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.GetCampaignResponse{
		HeadSeq: inspection.HeadSeq,
		State:   state,
	}, nil
}

// ListCampaigns returns the caller's recent campaign summaries.
func (s *Server) ListCampaigns(ctx context.Context, req *gamev1.ListCampaignsRequest) (*gamev1.ListCampaignsResponse, error) {
	svc, err := s.requireService()
	if err != nil {
		return nil, err
	}
	if err := requireRequest(req); err != nil {
		return nil, err
	}
	act, err := callerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	items, err := svc.ListCampaigns(ctx, act)
	if err != nil {
		return nil, mapDomainError(err)
	}
	response := &gamev1.ListCampaignsResponse{
		Campaigns: make([]*gamev1.CampaignSummary, 0, len(items)),
	}
	for _, item := range items {
		response.Campaigns = append(response.Campaigns, protoCampaignSummary(item))
	}
	return response, nil
}

// GetCampaignPlayReadiness returns deterministic blockers preventing entry into PLAY.
func (s *Server) GetCampaignPlayReadiness(ctx context.Context, req *gamev1.GetCampaignPlayReadinessRequest) (*gamev1.GetCampaignPlayReadinessResponse, error) {
	svc, err := s.requireService()
	if err != nil {
		return nil, err
	}
	if err := requireRequest(req); err != nil {
		return nil, err
	}
	campaignID, err := requestCampaignID(req.GetCampaignId())
	if err != nil {
		return nil, invalidArgument(err)
	}
	act, err := callerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	report, err := svc.GetPlayReadiness(ctx, act, campaignID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	readinessProto, err := protoPlayReadiness(report)
	if err != nil {
		return nil, internalStatus(err)
	}
	return &gamev1.GetCampaignPlayReadinessResponse{Readiness: readinessProto}, nil
}
