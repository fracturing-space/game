package gamev1

import (
	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
)

func protoCampaignSummary(input service.CampaignSummary) *gamev1.CampaignSummary {
	return &gamev1.CampaignSummary{
		CampaignId:       input.CampaignID,
		Name:             input.Name,
		ReadyToPlay:      input.ReadyToPlay,
		HasAiBinding:     input.HasAIBinding,
		HasActiveSession: input.HasActiveSession,
	}
}

func protoState(state campaign.Snapshot) (*gamev1.State, error) {
	playState, err := protoCampaignPlayState(state.PlayState)
	if err != nil {
		return nil, err
	}
	characters := make([]*gamev1.Character, 0, len(state.Characters))
	for _, next := range state.Characters {
		characters = append(characters, &gamev1.Character{
			Id:            next.ID,
			ParticipantId: next.ParticipantID,
			Name:          next.Name,
		})
	}
	participants := make([]*gamev1.Participant, 0, len(state.Participants))
	for _, next := range state.Participants {
		access, err := protoAccess(next.Access)
		if err != nil {
			return nil, err
		}
		participants = append(participants, &gamev1.Participant{
			Id:     next.ID,
			Name:   next.Name,
			Access: access,
		})
	}
	activeSession, err := protoActiveSession(state.ActiveSession())
	if err != nil {
		return nil, err
	}
	scenes := make([]*gamev1.Scene, 0, len(state.Scenes))
	for _, next := range state.Scenes {
		scenes = append(scenes, protoScene(next))
	}
	return &gamev1.State{
		Id:            state.ID,
		Name:          state.Name,
		PlayState:     playState,
		Characters:    characters,
		Participants:  participants,
		AiAgentId:     state.AIAgentID,
		ActiveSession: activeSession,
		ActiveSceneId: state.ActiveSceneID,
		Scenes:        scenes,
	}, nil
}

func protoActiveSession(input *session.Record) (*gamev1.Session, error) {
	if input == nil {
		return nil, nil
	}
	protoSession, err := protoSession(input.ID, input.Name, input.Status, input.CharacterControllers)
	if err != nil {
		return nil, err
	}
	return protoSession, nil
}

func protoScene(input scene.Record) *gamev1.Scene {
	return &gamev1.Scene{
		Id:           input.ID,
		SessionId:    input.SessionID,
		Name:         input.Name,
		Active:       input.Active,
		Ended:        input.Ended,
		CharacterIds: append([]string{}, input.CharacterIDs...),
	}
}
