package gamev1

import (
	"testing"

	gamev1pb "github.com/fracturing-space/game/api/gen/go/game/v1"
)

func TestWriteHandlersCampaignLifecycleSmoke(t *testing.T) {
	t.Parallel()

	server := mustServer(t, mustRealService(t))
	ownerCtx := inboundContext("subject-owner")
	playerCtx := inboundContext("subject-player-2")

	createResp, err := server.CreateCampaign(ownerCtx, &gamev1pb.CreateCampaignRequest{Name: "Autumn Twilight", OwnerName: "Owner"})
	if err != nil {
		t.Fatalf("CreateCampaign() error = %v", err)
	}
	campaignID := createResp.GetCampaignId()
	if campaignID == "" {
		t.Fatal("campaign id = empty, want generated id")
	}

	if _, err := server.UpdateCampaign(ownerCtx, &gamev1pb.UpdateCampaignRequest{CampaignId: campaignID, Name: "Autumn Eclipse"}); err != nil {
		t.Fatalf("UpdateCampaign() error = %v", err)
	}

	campaignResp, err := server.GetCampaign(ownerCtx, &gamev1pb.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		t.Fatalf("GetCampaign() error = %v", err)
	}
	ownerParticipantID := ""
	for _, next := range campaignResp.GetState().GetParticipants() {
		if next.GetAccess() == gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_OWNER {
			ownerParticipantID = next.GetId()
		}
	}
	if ownerParticipantID == "" {
		t.Fatal("participant id = empty, want owner participant")
	}

	createParticipantResp, err := server.CreateParticipant(ownerCtx, &gamev1pb.CreateParticipantRequest{
		CampaignId: campaignID,
		Name:       "Zoe", Access: gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_MEMBER})
	if err != nil {
		t.Fatalf("CreateParticipant() error = %v", err)
	}
	participantID := createParticipantResp.GetParticipantId()
	if participantID == "" {
		t.Fatal("participant id = empty, want generated id")
	}

	if _, err := server.UpdateParticipant(ownerCtx, &gamev1pb.UpdateParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
		Name:          "Zoe Prime", Access: gamev1pb.ParticipantAccess_PARTICIPANT_ACCESS_MEMBER}); err != nil {
		t.Fatalf("UpdateParticipant() error = %v", err)
	}

	if _, err := server.BindParticipant(playerCtx, &gamev1pb.BindParticipantRequest{CampaignId: campaignID, ParticipantId: participantID}); err != nil {
		t.Fatalf("BindParticipant() error = %v", err)
	}
	if _, err := server.UnbindParticipant(playerCtx, &gamev1pb.UnbindParticipantRequest{CampaignId: campaignID, ParticipantId: participantID}); err != nil {
		t.Fatalf("UnbindParticipant() error = %v", err)
	}
	if _, err := server.DeleteParticipant(ownerCtx, &gamev1pb.DeleteParticipantRequest{CampaignId: campaignID, ParticipantId: participantID}); err != nil {
		t.Fatalf("DeleteParticipant() error = %v", err)
	}

	if _, err := server.SetCampaignAIBinding(ownerCtx, &gamev1pb.SetCampaignAIBindingRequest{CampaignId: campaignID, AiAgentId: "agent-7"}); err != nil {
		t.Fatalf("SetCampaignAIBinding() error = %v", err)
	}

	createCharacterResp, err := server.CreateCharacter(ownerCtx, &gamev1pb.CreateCharacterRequest{
		CampaignId:    campaignID,
		ParticipantId: ownerParticipantID,
		Name:          "Luna"})
	if err != nil {
		t.Fatalf("CreateCharacter() error = %v", err)
	}
	characterID := createCharacterResp.GetCharacterId()
	if characterID == "" {
		t.Fatal("character id = empty, want generated id")
	}

	if _, err := server.UpdateCharacter(ownerCtx, &gamev1pb.UpdateCharacterRequest{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: ownerParticipantID,
		Name:          "Luna Prime"}); err != nil {
		t.Fatalf("UpdateCharacter() error = %v", err)
	}

	if _, err := server.BeginPlay(ownerCtx, &gamev1pb.BeginPlayRequest{CampaignId: campaignID}); err != nil {
		t.Fatalf("BeginPlay() error = %v", err)
	}
	campaignResp, err = server.GetCampaign(ownerCtx, &gamev1pb.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		t.Fatalf("GetCampaign(active) error = %v", err)
	}
	if campaignResp.GetState().GetActiveSession() == nil {
		t.Fatal("GetCampaign(active).active_session = nil, want session")
	}
	if got := campaignResp.GetState().GetActiveSceneId(); got != "" {
		t.Fatalf("GetCampaign(active).active_scene_id = %q, want empty", got)
	}
	if _, err := server.EndPlay(ownerCtx, &gamev1pb.EndPlayRequest{CampaignId: campaignID}); err != nil {
		t.Fatalf("EndPlay() error = %v", err)
	}
	if _, err := server.ClearCampaignAIBinding(ownerCtx, &gamev1pb.ClearCampaignAIBindingRequest{CampaignId: campaignID}); err != nil {
		t.Fatalf("ClearCampaignAIBinding() error = %v", err)
	}
	if _, err := server.DeleteCharacter(ownerCtx, &gamev1pb.DeleteCharacterRequest{CampaignId: campaignID, CharacterId: characterID}); err != nil {
		t.Fatalf("DeleteCharacter() error = %v", err)
	}
}
