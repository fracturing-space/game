package gamev1

import (
	"context"
	"fmt"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/service"
)

func executeCommand[T any](s *Server, ctx context.Context, req *T, build func(*T) (command.Envelope, error)) (service.Result, error) {
	svc, err := s.requireService()
	if err != nil {
		return service.Result{}, err
	}
	if err := requireRequest(req); err != nil {
		return service.Result{}, err
	}
	envelope, err := build(req)
	if err != nil {
		return service.Result{}, invalidArgument(err)
	}
	act, err := callerFromContext(ctx)
	if err != nil {
		return service.Result{}, err
	}
	result, err := svc.CommitCommand(ctx, act, envelope)
	if err != nil {
		return service.Result{}, mapDomainError(err)
	}
	return result, nil
}

func requestCampaignID(campaignID string) (string, error) {
	if err := canonical.ValidateExact(campaignID, "campaign id", fmt.Errorf); err != nil {
		return "", err
	}
	return campaignID, nil
}

func domainAccess(access gamev1.ParticipantAccess) (participant.Access, error) {
	switch access {
	case gamev1.ParticipantAccess_PARTICIPANT_ACCESS_OWNER:
		return participant.AccessOwner, nil
	case gamev1.ParticipantAccess_PARTICIPANT_ACCESS_MEMBER:
		return participant.AccessMember, nil
	default:
		return "", fmt.Errorf("participant access is required")
	}
}
