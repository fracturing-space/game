package gamev1

import (
	"fmt"
	"maps"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/readiness"
	"github.com/fracturing-space/game/internal/service"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const playReadinessErrorDomain = "game.v1.play_readiness"
const playReadinessErrorSubject = "campaign.mode"

func isPlayReadinessRejection(err error) bool {
	_, ok := readiness.AsRejection(err)
	return ok
}

func mapPlayReadinessError(err error) error {
	rejection, ok := readiness.AsRejection(err)
	if !ok {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	st := status.New(codes.FailedPrecondition, rejection.Message)
	detailed, detailErr := st.WithDetails(
		&errdetails.ErrorInfo{
			Reason: rejection.Code,
			Domain: playReadinessErrorDomain,
		},
		&errdetails.PreconditionFailure{
			Violations: []*errdetails.PreconditionFailure_Violation{{
				Type:        rejection.Code,
				Subject:     playReadinessErrorSubject,
				Description: rejection.Message,
			}},
		},
	)
	if detailErr != nil {
		return status.Error(codes.Internal, detailErr.Error())
	}
	return detailed.Err()
}

func protoPlayReadiness(input service.PlayReadiness) (*gamev1.CampaignPlayReadiness, error) {
	blockers := make([]*gamev1.CampaignPlayReadinessBlocker, 0, len(input.Blockers))
	for _, blocker := range input.Blockers {
		action, err := protoPlayReadinessAction(blocker.Action)
		if err != nil {
			return nil, err
		}
		metadata := make(map[string]string, len(blocker.Metadata))
		maps.Copy(metadata, blocker.Metadata)
		blockers = append(blockers, &gamev1.CampaignPlayReadinessBlocker{
			Code:     blocker.Code,
			Message:  blocker.Message,
			Metadata: metadata,
			Action:   action,
		})
	}
	return &gamev1.CampaignPlayReadiness{
		Ready:    input.Ready(),
		Blockers: blockers,
	}, nil
}

func protoPlayReadinessAction(input readiness.Action) (*gamev1.CampaignPlayReadinessAction, error) {
	kind, err := protoPlayReadinessResolutionKind(input.ResolutionKind)
	if err != nil {
		return nil, err
	}
	return &gamev1.CampaignPlayReadinessAction{
		ResponsibleParticipantIds: append([]string{}, input.ResponsibleParticipantIDs...),
		ResolutionKind:            kind,
		TargetParticipantId:       input.TargetParticipantID,
	}, nil
}

func protoPlayReadinessResolutionKind(kind readiness.ResolutionKind) (gamev1.CampaignPlayReadinessResolutionKind, error) {
	switch kind {
	case readiness.ResolutionKindUnspecified:
		return gamev1.CampaignPlayReadinessResolutionKind_CAMPAIGN_PLAY_READINESS_RESOLUTION_KIND_UNSPECIFIED, nil
	case readiness.ResolutionKindConfigureAIAgent:
		return gamev1.CampaignPlayReadinessResolutionKind_CAMPAIGN_PLAY_READINESS_RESOLUTION_KIND_CONFIGURE_AI_AGENT, nil
	case readiness.ResolutionKindManageParticipants:
		return gamev1.CampaignPlayReadinessResolutionKind_CAMPAIGN_PLAY_READINESS_RESOLUTION_KIND_MANAGE_PARTICIPANTS, nil
	case readiness.ResolutionKindInvitePlayer:
		return gamev1.CampaignPlayReadinessResolutionKind_CAMPAIGN_PLAY_READINESS_RESOLUTION_KIND_INVITE_PLAYER, nil
	case readiness.ResolutionKindCreateCharacter:
		return gamev1.CampaignPlayReadinessResolutionKind_CAMPAIGN_PLAY_READINESS_RESOLUTION_KIND_CREATE_CHARACTER, nil
	default:
		return gamev1.CampaignPlayReadinessResolutionKind_CAMPAIGN_PLAY_READINESS_RESOLUTION_KIND_UNSPECIFIED, fmt.Errorf("campaign play readiness resolution kind is invalid: %s", kind)
	}
}
