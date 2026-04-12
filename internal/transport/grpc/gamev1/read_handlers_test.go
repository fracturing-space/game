package gamev1

import (
	"context"
	"testing"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/readiness"
	"github.com/fracturing-space/game/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type readTransportService struct {
	panicTransportService
	inspect   func(context.Context, caller.Caller, string) (service.Inspection, error)
	list      func(context.Context, caller.Caller) ([]service.CampaignSummary, error)
	readiness func(context.Context, caller.Caller, string) (service.PlayReadiness, error)
}

func (s *readTransportService) Inspect(ctx context.Context, act caller.Caller, campaignID string) (service.Inspection, error) {
	if s.inspect != nil {
		return s.inspect(ctx, act, campaignID)
	}
	return service.Inspection{}, nil
}

func (s *readTransportService) ListCampaigns(ctx context.Context, act caller.Caller) ([]service.CampaignSummary, error) {
	if s.list != nil {
		return s.list(ctx, act)
	}
	return nil, nil
}

func (s *readTransportService) GetPlayReadiness(ctx context.Context, act caller.Caller, campaignID string) (service.PlayReadiness, error) {
	if s.readiness != nil {
		return s.readiness(ctx, act, campaignID)
	}
	return service.PlayReadiness{}, nil
}

func TestGetCampaignHandler(t *testing.T) {
	t.Parallel()

	t.Run("rejects nil request", func(t *testing.T) {
		server := mustServer(t, &panicTransportService{})
		if _, err := server.GetCampaign(inboundContext("subject-1"), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("GetCampaign(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})

	t.Run("rejects blank campaign id", func(t *testing.T) {
		server := mustServer(t, &panicTransportService{})
		if _, err := server.GetCampaign(inboundContext("subject-1"), &gamev1pb.GetCampaignRequest{}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("GetCampaign(blank) code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})

	t.Run("maps denied reads", func(t *testing.T) {
		server := mustServer(t, &readTransportService{
			inspect: func(context.Context, caller.Caller, string) (service.Inspection, error) {
				return service.Inspection{}, &authz.DeniedError{Capability: authz.CapabilityReadCampaign, Reason: "nope"}
			},
		})
		if _, err := server.GetCampaign(inboundContext("subject-1"), &gamev1pb.GetCampaignRequest{CampaignId: "camp-1"}); status.Code(err) != codes.PermissionDenied {
			t.Fatalf("GetCampaign(denied) code = %v, want %v", status.Code(err), codes.PermissionDenied)
		}
	})

	t.Run("returns inspection state", func(t *testing.T) {
		server := mustServer(t, &readTransportService{
			inspect: func(_ context.Context, act caller.Caller, campaignID string) (service.Inspection, error) {
				if got, want := act.SubjectID, "subject-1"; got != want {
					t.Fatalf("caller subject id = %q, want %q", got, want)
				}
				return service.Inspection{
					HeadSeq: 4,
					State:   campaign.Snapshot{ID: campaignID, Name: "Autumn Twilight", PlayState: campaign.PlayStateSetup},
				}, nil
			},
		})
		resp, err := server.GetCampaign(inboundContext("subject-1"), &gamev1pb.GetCampaignRequest{CampaignId: "camp-1"})
		if err != nil {
			t.Fatalf("GetCampaign() error = %v", err)
		}
		if got, want := resp.GetHeadSeq(), uint64(4); got != want {
			t.Fatalf("head seq = %d, want %d", got, want)
		}
		if got, want := resp.GetState().GetId(), "camp-1"; got != want {
			t.Fatalf("state id = %q, want %q", got, want)
		}
	})
}

func TestListCampaignsHandler(t *testing.T) {
	t.Parallel()

	t.Run("rejects nil request", func(t *testing.T) {
		server := mustServer(t, &panicTransportService{})
		if _, err := server.ListCampaigns(inboundContext("subject-1"), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("ListCampaigns(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})

	server := mustServer(t, &readTransportService{
		list: func(_ context.Context, act caller.Caller) ([]service.CampaignSummary, error) {
			if got, want := act.SubjectID, "subject-1"; got != want {
				t.Fatalf("caller subject id = %q, want %q", got, want)
			}
			return []service.CampaignSummary{{
				CampaignID:       "camp-1",
				Name:             "Autumn Twilight",
				ReadyToPlay:      true,
				HasAIBinding:     true,
				HasActiveSession: true,
			}}, nil
		},
	})
	resp, err := server.ListCampaigns(inboundContext("subject-1"), &gamev1pb.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns() error = %v", err)
	}
	if got, want := len(resp.GetCampaigns()), 1; got != want {
		t.Fatalf("campaigns len = %d, want %d", got, want)
	}
	if got, want := resp.GetCampaigns()[0].GetCampaignId(), "camp-1"; got != want {
		t.Fatalf("campaign id = %q, want %q", got, want)
	}
}

func TestGetCampaignPlayReadinessHandler(t *testing.T) {
	t.Parallel()

	t.Run("rejects nil request", func(t *testing.T) {
		server := mustServer(t, &panicTransportService{})
		if _, err := server.GetCampaignPlayReadiness(inboundContext("subject-1"), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("GetCampaignPlayReadiness(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
	})

	t.Run("returns mapped play-readiness details", func(t *testing.T) {
		server := mustServer(t, &readTransportService{
			readiness: func(context.Context, caller.Caller, string) (service.PlayReadiness, error) {
				return service.PlayReadiness{}, &readiness.Rejection{
					Code:    readiness.RejectionCodePlayReadinessAIAgentRequired,
					Message: "AI agent binding is required before entering PLAY",
				}
			},
		})
		_, err := server.GetCampaignPlayReadiness(inboundContext("subject-1"), &gamev1pb.GetCampaignPlayReadinessRequest{CampaignId: "camp-1"})
		assertPlayReadinessStatusDetails(t, err, readiness.RejectionCodePlayReadinessAIAgentRequired, "AI agent binding is required before entering PLAY")
	})

	t.Run("returns readiness report", func(t *testing.T) {
		server := mustServer(t, &readTransportService{
			readiness: func(context.Context, caller.Caller, string) (service.PlayReadiness, error) {
				return service.PlayReadiness{
					Blockers: []readiness.Blocker{{
						Code:    readiness.RejectionCodePlayReadinessPlayerCharacterRequired,
						Message: "character required",
						Action: readiness.Action{
							ResponsibleParticipantIDs: []string{"part-1"},
							ResolutionKind:            readiness.ResolutionKindCreateCharacter,
							TargetParticipantID:       "part-1",
						},
					}},
				}, nil
			},
		})
		resp, err := server.GetCampaignPlayReadiness(inboundContext("subject-1"), &gamev1pb.GetCampaignPlayReadinessRequest{CampaignId: "camp-1"})
		if err != nil {
			t.Fatalf("GetCampaignPlayReadiness() error = %v", err)
		}
		if got, want := len(resp.GetReadiness().GetBlockers()), 1; got != want {
			t.Fatalf("blockers len = %d, want %d", got, want)
		}
		if got, want := resp.GetReadiness().GetBlockers()[0].GetAction().GetResolutionKind(), gamev1pb.CampaignPlayReadinessResolutionKind_CAMPAIGN_PLAY_READINESS_RESOLUTION_KIND_CREATE_CHARACTER; got != want {
			t.Fatalf("resolution kind = %v, want %v", got, want)
		}
	})
}
