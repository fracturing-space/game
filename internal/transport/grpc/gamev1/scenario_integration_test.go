package gamev1

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/service"
	memorystorage "github.com/fracturing-space/game/internal/storage/memory"
	sqlitestorage "github.com/fracturing-space/game/internal/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestScenarioCampaignLifecycleOverGRPC(t *testing.T) {
	t.Parallel()

	const ownerCaller = "subject-owner"

	runScenario(t, scenarioSpec{
		name: "campaign lifecycle over grpc",
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
				name:   "list campaigns",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.ListCampaigns(ctx, &gamev1pb.ListCampaignsRequest{})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.ListCampaignsResponse](t, got)
					if got, want := len(resp.GetCampaigns()), 1; got != want {
						t.Fatalf("ListCampaigns() len = %d, want %d", got, want)
					}
					if got, want := resp.GetCampaigns()[0].GetCampaignId(), runtime.ref("campaign_id"); got != want {
						t.Fatalf("ListCampaigns()[0].campaign_id = %q, want %q", got, want)
					}
				},
			},
			{
				name:   "read initial readiness",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.GetCampaignPlayReadiness(ctx, &gamev1pb.GetCampaignPlayReadinessRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.GetCampaignPlayReadinessResponse](t, got)
					if resp.GetReadiness().GetReady() {
						t.Fatal("GetCampaignPlayReadiness().ready = true, want false")
					}
					assertReadinessCodes(t, resp,
						"PLAY_READINESS_AI_AGENT_REQUIRED",
						"PLAY_READINESS_PLAYER_CHARACTER_REQUIRED",
					)
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
					if got, want := resp.GetCampaignId(), runtime.ref("campaign_id"); got != want {
						t.Fatalf("SetCampaignAIBinding().campaign_id = %q, want %q", got, want)
					}
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
				name:   "read ready readiness",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return runtime.harness.client.GetCampaignPlayReadiness(ctx, &gamev1pb.GetCampaignPlayReadinessRequest{
						CampaignId: runtime.ref("campaign_id"),
					})
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					resp := scenarioResultAs[*gamev1pb.GetCampaignPlayReadinessResponse](t, got)
					if !resp.GetReadiness().GetReady() {
						t.Fatalf("GetCampaignPlayReadiness(ready).blockers = %v, want ready report", resp.GetReadiness().GetBlockers())
					}
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
					if got, want := resp.GetState().GetPlayState(), gamev1pb.CampaignPlayState_CAMPAIGN_PLAY_STATE_ACTIVE; got != want {
						t.Fatalf("GetCampaign().play_state = %v, want %v", got, want)
					}
					if got, want := resp.GetState().GetAiAgentId(), "agent-7"; got != want {
						t.Fatalf("GetCampaign().ai_agent_id = %q, want %q", got, want)
					}
					if got := resp.GetState().GetActiveSession(); got == nil {
						t.Fatal("GetCampaign().active_session = nil, want session")
					}
					runtime.setRef("session_id", resp.GetState().GetActiveSession().GetId())
					if got, want := resp.GetState().GetActiveSession().GetName(), "Session 1"; got != want {
						t.Fatalf("GetCampaign().active_session.name = %q, want %q", got, want)
					}
					if got := resp.GetState().GetActiveSceneId(); got != "" {
						t.Fatalf("GetCampaign().active_scene_id = %q, want empty", got)
					}
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
				name:   "open committed stream",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					streamCtx, cancel := context.WithCancel(ctx)
					stream, err := runtime.harness.client.StreamCampaignEvents(streamCtx, &gamev1pb.StreamCampaignEventsRequest{
						CampaignId: runtime.ref("campaign_id"),
						AfterSeq:   0,
					})
					if err != nil {
						cancel()
						return nil, err
					}
					runtime.setStream("campaign", scenarioStream{client: stream, cancel: cancel})
					return nil, nil
				},
			},
			{
				name:   "assert committed event order",
				caller: ownerCaller,
				action: func(ctx context.Context, runtime *scenarioRuntime) (any, error) {
					return recvEventTypes(runtime.stream("campaign").client, 8)
				},
				assert: func(t *testing.T, runtime *scenarioRuntime, got any) {
					gotTypes := scenarioResultAs[[]string](t, got)
					wantTypes := []string{
						"campaign.created",
						"participant.joined",
						"campaign.ai_bound",
						"character.created",
						"session.started",
						"campaign.play.began",
						"session.ended",
						"campaign.play.ended",
					}
					if got, want := len(gotTypes), len(wantTypes); got != want {
						t.Fatalf("event types len = %d, want %d", got, want)
					}
					for idx, want := range wantTypes {
						if got := gotTypes[idx]; got != want {
							t.Fatalf("event %d type = %q, want %q", idx, got, want)
						}
					}
				},
			},
		},
	})
}

func TestScenarioAuthorizationChangesAfterSeatBind(t *testing.T) {
	t.Parallel()

	harness := newScenarioHarness(t)
	ownerCtx := outgoingSubjectContext("subject-owner")
	playerCtx := outgoingSubjectContext("subject-player-2")

	createResp, err := harness.client.CreateCampaign(ownerCtx, &gamev1pb.CreateCampaignRequest{
		Name:      "Autumn Twilight",
		OwnerName: "Owner",
	})
	if err != nil {
		t.Fatalf("CreateCampaign() error = %v", err)
	}
	campaignID := createResp.GetCampaignId()

	if _, err := harness.client.GetCampaign(playerCtx, &gamev1pb.GetCampaignRequest{CampaignId: campaignID}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("GetCampaign(unjoined) error code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}

	createParticipantResp, err := harness.client.CreateParticipant(ownerCtx, &gamev1pb.CreateParticipantRequest{
		CampaignId: campaignID,
		Name:       "Zoe", Access: gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_MEMBER})
	if err != nil {
		t.Fatalf("CreateParticipant() error = %v", err)
	}
	if createParticipantResp.GetParticipantId() == "" {
		t.Fatal("CreateParticipant().participant_id = empty, want generated id")
	}

	if _, err := harness.client.GetCampaign(playerCtx, &gamev1pb.GetCampaignRequest{CampaignId: campaignID}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("GetCampaign(unbound participant) error code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}

	bindResp, err := harness.client.BindParticipant(playerCtx, &gamev1pb.BindParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: createParticipantResp.GetParticipantId(),
	})
	if err != nil {
		t.Fatalf("BindParticipant() error = %v", err)
	}
	if got, want := bindResp.GetParticipantId(), createParticipantResp.GetParticipantId(); got != want {
		t.Fatalf("BindParticipant().participant_id = %q, want %q", got, want)
	}

	campaignResp, err := harness.client.GetCampaign(playerCtx, &gamev1pb.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		t.Fatalf("GetCampaign(bound) error = %v", err)
	}
	if got, want := len(campaignResp.GetState().GetParticipants()), 2; got != want {
		t.Fatalf("GetCampaign(bound).participants len = %d, want %d", got, want)
	}

	listResp, err := harness.client.ListCampaigns(playerCtx, &gamev1pb.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("ListCampaigns(bound) error = %v", err)
	}
	if got, want := len(listResp.GetCampaigns()), 1; got != want {
		t.Fatalf("ListCampaigns(bound) len = %d, want %d", got, want)
	}
	if got, want := listResp.GetCampaigns()[0].GetCampaignId(), campaignID; got != want {
		t.Fatalf("ListCampaigns(bound)[0].campaign_id = %q, want %q", got, want)
	}
}

func assertReadinessCodes(t *testing.T, resp *gamev1pb.GetCampaignPlayReadinessResponse, want ...string) {
	t.Helper()

	if got, wantLen := len(resp.GetReadiness().GetBlockers()), len(want); got != wantLen {
		t.Fatalf("GetCampaignPlayReadiness().blockers len = %d, want %d", got, wantLen)
	}
	for idx, code := range want {
		if got := resp.GetReadiness().GetBlockers()[idx].GetCode(); got != code {
			t.Fatalf("blocker %d code = %q, want %q", idx, got, code)
		}
	}
}

func ownerParticipantID(t *testing.T, resp *gamev1pb.GetCampaignResponse) string {
	t.Helper()

	for _, next := range resp.GetState().GetParticipants() {
		if next.GetAccess() == gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_OWNER {
			if next.GetId() == "" {
				t.Fatal("participant id = empty, want owner participant")
			}
			return next.GetId()
		}
	}
	t.Fatal("participant id = empty, want owner participant")
	return ""
}

func recvEventTypes(stream gamev1pb.GameService_StreamCampaignEventsClient, count int) ([]string, error) {
	gotTypes := make([]string, 0, count)
	for range count {
		event, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		gotTypes = append(gotTypes, event.GetType())
	}
	return gotTypes, nil
}

type scenarioHarness struct {
	client      gamev1pb.GameServiceClient
	conn        *grpc.ClientConn
	server      *grpc.Server
	listener    *bufconn.Listener
	projections service.ProjectionStore
	close       func()
	reopen      func(*testing.T) scenarioHarness
}

func newScenarioHarness(t *testing.T) scenarioHarness {
	t.Helper()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	ids := newSequentialIDAllocator()
	stores := memorystorage.NewBundle()
	return openScenarioHarnessWithService(t, manifest, ids, stores.Journal, stores.ProjectionStore, stores.ArtifactStore, nil, nil)
}

func newSQLiteScenarioHarness(t *testing.T) scenarioHarness {
	t.Helper()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	ids := newSequentialIDAllocator()
	paths := sqlitestorage.Paths{
		EventsDBPath:      filepath.Join(t.TempDir(), "events.db"),
		ProjectionsDBPath: filepath.Join(t.TempDir(), "projections.db"),
		ArtifactsDBPath:   filepath.Join(t.TempDir(), "artifacts.db"),
	}
	return openSQLiteScenarioHarness(t, manifest, ids, paths)
}

func openSQLiteScenarioHarness(t *testing.T, manifest *service.Manifest, ids *sequentialIDAllocator, paths sqlitestorage.Paths) scenarioHarness {
	t.Helper()

	stores, err := sqlitestorage.Open(manifest, paths)
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	return openScenarioHarnessWithService(
		t,
		manifest,
		ids,
		stores.Journal,
		stores.ProjectionStore,
		stores.ArtifactStore,
		func() {
			_ = stores.Close()
		},
		func(t *testing.T) scenarioHarness {
			return openSQLiteScenarioHarness(t, manifest, ids, paths)
		},
	)
}

func openScenarioHarnessWithService(
	t *testing.T,
	manifest *service.Manifest,
	ids *sequentialIDAllocator,
	journal service.Journal,
	projections service.ProjectionStore,
	artifacts service.ArtifactStore,
	closeStores func(),
	reopen func(*testing.T) scenarioHarness,
) scenarioHarness {
	t.Helper()

	svc, err := service.New(service.Config{
		Manifest:        manifest,
		IDs:             ids,
		Journal:         journal,
		ProjectionStore: projections,
		ArtifactStore:   artifacts,
	})
	if err != nil {
		t.Fatalf("service.New() error = %v", err)
	}
	handler, err := NewServer(svc)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	gamev1pb.RegisterGameServiceServer(server, handler)
	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		server.Stop()
		_ = listener.Close()
		if closeStores != nil {
			closeStores()
		}
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	closeHarness := func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
		if closeStores != nil {
			closeStores()
		}
	}
	t.Cleanup(closeHarness)

	return scenarioHarness{
		client:      gamev1pb.NewGameServiceClient(conn),
		conn:        conn,
		server:      server,
		listener:    listener,
		projections: projections,
		close:       closeHarness,
		reopen:      reopen,
	}
}

func outgoingSubjectContext(subjectID string) context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(subjectIDHeader, subjectID))
}
