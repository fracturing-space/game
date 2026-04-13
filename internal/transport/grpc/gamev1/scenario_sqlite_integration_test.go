package gamev1

import (
	"context"
	"testing"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	campaigndomain "github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/service"
)

func TestScenarioCampaignLifecyclePersistsProjectionOverSQLite(t *testing.T) {
	t.Parallel()

	const ownerCaller = "subject-owner"

	runScenario(t, scenarioSpec{
		name:       "campaign lifecycle persists projection over sqlite",
		newHarness: newSQLiteScenarioHarness,
		steps: []scenarioStep{
			{
				name:   "create campaign",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.CreateCampaign(ctx, &gamev1pb.CreateCampaignRequest{
						Name:      "Autumn Twilight",
						OwnerName: "Owner",
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.CreateCampaignResponse](t, got)
					runtime.setRef("campaign_id", resp.GetCampaignId())
				},
			},
			{
				name:   "set ai binding",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.SetCampaignAIBinding(ctx, &gamev1pb.SetCampaignAIBindingRequest{
						CampaignId: runtime.ref("campaign_id"),
						AiAgentId:  "agent-7",
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.SetCampaignAIBindingResponse](t, got)
					if got, want := resp.GetAiAgentId(), "agent-7"; got != want {
						t.Fatalf("SetCampaignAIBinding().ai_agent_id = %q, want %q", got, want)
					}
				},
			},
			{
				name:   "get campaign and capture owner participant",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.GetCampaign(ctx, &gamev1pb.GetCampaignRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.GetCampaignResponse](t, got)
					runtime.setRef("participant_id", ownerParticipantID(t, resp))
				},
			},
			{
				name:   "create character",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.CreateCharacter(ctx, &gamev1pb.CreateCharacterRequest{
						CampaignId:    runtime.ref("campaign_id"),
						ParticipantId: runtime.ref("participant_id"),
						Name:          "Luna"})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.CreateCharacterResponse](t, got)
					runtime.setRef("character_id", resp.GetCharacterId())
				},
			},
			{
				name:   "begin play",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.BeginPlay(ctx, &gamev1pb.BeginPlayRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.BeginPlayResponse](t, got)
					if got, want := resp.GetCampaignId(), runtime.ref("campaign_id"); got != want {
						t.Fatalf("BeginPlay().campaign_id = %q, want %q", got, want)
					}
				},
			},
			{
				name:   "get active campaign",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.GetCampaign(ctx, &gamev1pb.GetCampaignRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.GetCampaignResponse](t, got)
					runtime.setRef("session_id", resp.GetState().GetActiveSession().GetId())
					assertActiveCampaignResponse(t, resp, runtime)
				},
			},
			{
				name:   "assert persisted active projection",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					snapshot, ok, err := runtime.projections().GetProjection(ctx, runtime.ref("campaign_id"))
					if err != nil {
						return nil, err
					}
					return projectionLookup{snapshot: snapshot, ok: ok}, nil
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					projection := scenarioResultAs[projectionLookup](t, got)
					if !projection.ok {
						t.Fatal("GetProjection(active) = missing, want stored projection")
					}
					if got, want := projection.snapshot.HeadSeq, uint64(6); got != want {
						t.Fatalf("GetProjection(active).HeadSeq = %d, want %d", got, want)
					}
					assertActiveProjectionState(t, projection.snapshot.State, runtime)
				},
			},
			{
				name:   "assert active watermark",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					watermark, ok, err := runtime.projections().GetWatermark(ctx, runtime.ref("campaign_id"))
					if err != nil {
						return nil, err
					}
					return watermarkLookup{value: watermark, ok: ok}, nil
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					watermark := scenarioResultAs[watermarkLookup](t, got)
					if !watermark.ok {
						t.Fatal("GetWatermark(active) = missing, want stored watermark")
					}
					if got, want := watermark.value.AppliedSeq, uint64(6); got != want {
						t.Fatalf("GetWatermark(active).AppliedSeq = %d, want %d", got, want)
					}
					if got, want := watermark.value.ExpectedNextSeq, uint64(7); got != want {
						t.Fatalf("GetWatermark(active).ExpectedNextSeq = %d, want %d", got, want)
					}
				},
			},
			{
				name:   "restart sqlite harness",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					runtime.restartHarness()
					return nil, nil
				},
			},
			{
				name:   "get active campaign after restart",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.GetCampaign(ctx, &gamev1pb.GetCampaignRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					assertActiveCampaignResponse(t, scenarioResultAs[*gamev1pb.GetCampaignResponse](t, got), runtime)
				},
			},
			{
				name:   "end play",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.EndPlay(ctx, &gamev1pb.EndPlayRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.EndPlayResponse](t, got)
					if got, want := resp.GetCampaignId(), runtime.ref("campaign_id"); got != want {
						t.Fatalf("EndPlay().campaign_id = %q, want %q", got, want)
					}
				},
			},
			{
				name:   "get setup campaign",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.GetCampaign(ctx, &gamev1pb.GetCampaignRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.GetCampaignResponse](t, got)
					if got, want := resp.GetState().GetPlayState(), gamev1pb.CampaignPlayState_CAMPAIGN_PLAY_STATE_SETUP; got != want {
						t.Fatalf("GetCampaign(after end).play_state = %v, want %v", got, want)
					}
					if resp.GetState().GetActiveSession() != nil {
						t.Fatal("GetCampaign(after end).active_session != nil, want nil")
					}
					if got := resp.GetState().GetActiveSceneId(); got != "" {
						t.Fatalf("GetCampaign(after end).active_scene_id = %q, want empty", got)
					}
				},
			},
			{
				name:   "assert persisted ended projection",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					snapshot, ok, err := runtime.projections().GetProjection(ctx, runtime.ref("campaign_id"))
					if err != nil {
						return nil, err
					}
					return projectionLookup{snapshot: snapshot, ok: ok}, nil
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					projection := scenarioResultAs[projectionLookup](t, got)
					if !projection.ok {
						t.Fatal("GetProjection(ended) = missing, want stored projection")
					}
					if got, want := projection.snapshot.HeadSeq, uint64(8); got != want {
						t.Fatalf("GetProjection(ended).HeadSeq = %d, want %d", got, want)
					}
					if got, want := projection.snapshot.State.PlayState, campaigndomain.PlayStateSetup; got != want {
						t.Fatalf("GetProjection(ended).State.PlayState = %q, want %q", got, want)
					}
					if projection.snapshot.State.ActiveSession() != nil {
						t.Fatal("GetProjection(ended).State.ActiveSession != nil, want nil")
					}
					if got := projection.snapshot.State.ActiveSceneID; got != "" {
						t.Fatalf("GetProjection(ended).State.ActiveSceneID = %q, want empty", got)
					}
				},
			},
			{
				name:   "assert ended watermark",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					watermark, ok, err := runtime.projections().GetWatermark(ctx, runtime.ref("campaign_id"))
					if err != nil {
						return nil, err
					}
					return watermarkLookup{value: watermark, ok: ok}, nil
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					watermark := scenarioResultAs[watermarkLookup](t, got)
					if !watermark.ok {
						t.Fatal("GetWatermark(ended) = missing, want stored watermark")
					}
					if got, want := watermark.value.AppliedSeq, uint64(8); got != want {
						t.Fatalf("GetWatermark(ended).AppliedSeq = %d, want %d", got, want)
					}
					if got, want := watermark.value.ExpectedNextSeq, uint64(9); got != want {
						t.Fatalf("GetWatermark(ended).ExpectedNextSeq = %d, want %d", got, want)
					}
				},
			},
			{
				name:   "list campaigns after end",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.ListCampaigns(ctx, &gamev1pb.ListCampaignsRequest{})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.ListCampaignsResponse](t, got)
					if got, want := len(resp.GetCampaigns()), 1; got != want {
						t.Fatalf("ListCampaigns(after end) len = %d, want %d", got, want)
					}
					summary := resp.GetCampaigns()[0]
					if got, want := summary.GetCampaignId(), runtime.ref("campaign_id"); got != want {
						t.Fatalf("ListCampaigns(after end)[0].campaign_id = %q, want %q", got, want)
					}
					if !summary.GetHasAiBinding() {
						t.Fatal("ListCampaigns(after end)[0].has_ai_binding = false, want true")
					}
					if summary.GetHasActiveSession() {
						t.Fatal("ListCampaigns(after end)[0].has_active_session = true, want false")
					}
					if !summary.GetReadyToPlay() {
						t.Fatal("ListCampaigns(after end)[0].ready_to_play = false, want true")
					}
				},
			},
		},
	})
}

type projectionLookup struct {
	snapshot service.ProjectionSnapshot
	ok       bool
}

type watermarkLookup struct {
	value service.ProjectionWatermark
	ok    bool
}

func assertActiveCampaignResponse(t *testing.T, resp *gamev1pb.GetCampaignResponse, runtime *scenarioRuntime) {
	t.Helper()

	if got, want := resp.GetState().GetPlayState(), gamev1pb.CampaignPlayState_CAMPAIGN_PLAY_STATE_ACTIVE; got != want {
		t.Fatalf("GetCampaign().play_state = %v, want %v", got, want)
	}
	if got, want := resp.GetState().GetAiAgentId(), "agent-7"; got != want {
		t.Fatalf("GetCampaign().ai_agent_id = %q, want %q", got, want)
	}
	if got := resp.GetState().GetActiveSession(); got == nil {
		t.Fatal("GetCampaign().active_session = nil, want session")
	}
	if got, want := resp.GetState().GetActiveSession().GetId(), runtime.ref("session_id"); got != want {
		t.Fatalf("GetCampaign().active_session.id = %q, want %q", got, want)
	}
	if got, want := resp.GetState().GetActiveSession().GetName(), "Session 1"; got != want {
		t.Fatalf("GetCampaign().active_session.name = %q, want %q", got, want)
	}
	if got := resp.GetState().GetActiveSceneId(); got != "" {
		t.Fatalf("GetCampaign().active_scene_id = %q, want empty", got)
	}
}

func assertActiveProjectionState(t *testing.T, state campaigndomain.State, runtime *scenarioRuntime) {
	t.Helper()

	if got, want := state.PlayState, campaigndomain.PlayStateActive; got != want {
		t.Fatalf("projection state play_state = %q, want %q", got, want)
	}
	if got, want := state.AIAgentID, "agent-7"; got != want {
		t.Fatalf("projection state ai_agent_id = %q, want %q", got, want)
	}
	if state.ActiveSession() == nil {
		t.Fatal("projection state active_session = nil, want session")
	}
	if got, want := state.ActiveSession().ID, runtime.ref("session_id"); got != want {
		t.Fatalf("projection state active_session.id = %q, want %q", got, want)
	}
	if got := state.ActiveSceneID; got != "" {
		t.Fatalf("projection state active_scene_id = %q, want empty", got)
	}
}
