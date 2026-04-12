package gamev1

import (
	"fmt"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type eventPayloadEncoder func(*gamev1.Event, event.Message) error

var eventPayloadEncoders = map[event.Type]eventPayloadEncoder{
	campaign.EventTypeCreated:     encodeCampaignCreatedPayload,
	campaign.EventTypeUpdated:     encodeCampaignUpdatedPayload,
	campaign.EventTypePlayBegan:   encodeCampaignPlayBeganPayload,
	campaign.EventTypePlayPaused:  encodeCampaignPlayPausedPayload,
	campaign.EventTypePlayResumed: encodeCampaignPlayResumedPayload,
	campaign.EventTypePlayEnded:   encodeCampaignPlayEndedPayload,
	campaign.EventTypeAIBound:     encodeCampaignAIBoundPayload,
	campaign.EventTypeAIUnbound:   encodeCampaignAIUnboundPayload,
	character.EventTypeCreated:    encodeCharacterCreatedPayload,
	character.EventTypeUpdated:    encodeCharacterUpdatedPayload,
	character.EventTypeDeleted:    encodeCharacterDeletedPayload,
	session.EventTypeStarted:      encodeSessionStartedPayload,
	session.EventTypeEnded:        encodeSessionEndedPayload,
	participant.EventTypeJoined:   encodeParticipantJoinedPayload,
	participant.EventTypeUpdated:  encodeParticipantUpdatedPayload,
	participant.EventTypeBound:    encodeParticipantBoundPayload,
	participant.EventTypeUnbound:  encodeParticipantUnboundPayload,
	participant.EventTypeLeft:     encodeParticipantLeftPayload,
	scene.EventTypeCreated:        encodeSceneCreatedPayload,
	scene.EventTypeActivated:      encodeSceneActivatedPayload,
	scene.EventTypeEnded:          encodeSceneEndedPayload,
	scene.EventTypeCastReplaced:   encodeSceneCastReplacedPayload,
}

func protoPlannedEvent(input event.Envelope) (*gamev1.Event, error) {
	return protoEvent(0, 0, input.Type(), input.CampaignID, input.Message, nil)
}

func protoStoredEvent(input event.Record) (*gamev1.Event, error) {
	return protoEvent(input.Seq, input.CommitSeq, input.Type(), input.CampaignID, input.Message, timestamppb.New(input.RecordedAt.UTC()))
}

func protoEvent(seq uint64, commitSeq uint64, typ event.Type, campaignID string, message event.Message, recordedAt *timestamppb.Timestamp) (*gamev1.Event, error) {
	next := &gamev1.Event{
		Seq:        seq,
		CommitSeq:  commitSeq,
		Type:       string(typ),
		CampaignId: campaignID,
		RecordedAt: recordedAt,
	}
	encoder, ok := eventPayloadEncoders[typ]
	if !ok {
		return nil, fmt.Errorf("unsupported event payload %T", message)
	}
	if err := encoder(next, message); err != nil {
		return nil, err
	}
	return next, nil
}

func encodeCampaignCreatedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(campaign.Created)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignCreated{
		CampaignCreated: &gamev1.CampaignCreated{Name: typed.Name},
	}
	return nil
}

func encodeCampaignUpdatedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(campaign.Updated)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignUpdated{
		CampaignUpdated: &gamev1.CampaignUpdated{Name: typed.Name},
	}
	return nil
}

func encodeCampaignPlayBeganPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(campaign.PlayBegan)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignPlayBegan{
		CampaignPlayBegan: &gamev1.CampaignPlayBegan{
			SessionId: typed.SessionID,
			SceneId:   typed.SceneID,
		},
	}
	return nil
}

func encodeCampaignPlayPausedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(campaign.PlayPaused)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignPlayPaused{
		CampaignPlayPaused: &gamev1.CampaignPlayPaused{
			SessionId: typed.SessionID,
			SceneId:   typed.SceneID,
			Reason:    typed.Reason,
		},
	}
	return nil
}

func encodeCampaignPlayResumedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(campaign.PlayResumed)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignPlayResumed{
		CampaignPlayResumed: &gamev1.CampaignPlayResumed{
			SessionId: typed.SessionID,
			SceneId:   typed.SceneID,
			Reason:    typed.Reason,
		},
	}
	return nil
}

func encodeCampaignPlayEndedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(campaign.PlayEnded)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignPlayEnded{
		CampaignPlayEnded: &gamev1.CampaignPlayEnded{
			SessionId: typed.SessionID,
			SceneId:   typed.SceneID,
		},
	}
	return nil
}

func encodeCampaignAIBoundPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(campaign.AIBound)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignAiBound{
		CampaignAiBound: &gamev1.CampaignAIBound{AiAgentId: typed.AIAgentID},
	}
	return nil
}

func encodeCampaignAIUnboundPayload(next *gamev1.Event, message event.Message) error {
	if _, ok := message.(campaign.AIUnbound); !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CampaignAiUnbound{
		CampaignAiUnbound: &gamev1.CampaignAIUnbound{},
	}
	return nil
}

func encodeCharacterCreatedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(character.Created)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CharacterCreated{
		CharacterCreated: &gamev1.CharacterCreated{
			CharacterId:   typed.CharacterID,
			ParticipantId: typed.ParticipantID,
			Name:          typed.Name,
		},
	}
	return nil
}

func encodeCharacterUpdatedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(character.Updated)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CharacterUpdated{
		CharacterUpdated: &gamev1.CharacterUpdated{
			CharacterId:   typed.CharacterID,
			ParticipantId: typed.ParticipantID,
			Name:          typed.Name,
		},
	}
	return nil
}

func encodeCharacterDeletedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(character.Deleted)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_CharacterDeleted{
		CharacterDeleted: &gamev1.CharacterDeleted{
			CharacterId: typed.CharacterID,
		},
	}
	return nil
}

func encodeSessionStartedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(session.Started)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_SessionStarted{
		SessionStarted: protoSessionPayload(typed.SessionID, typed.Name, typed.CharacterControllers),
	}
	return nil
}

func encodeSessionEndedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(session.Ended)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_SessionEnded{
		SessionEnded: &gamev1.SessionEnded{
			SessionId:            typed.SessionID,
			Name:                 typed.Name,
			CharacterControllers: protoSessionCharacterControllers(typed.CharacterControllers),
		},
	}
	return nil
}

func encodeParticipantJoinedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(participant.Joined)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	access, err := protoAccess(typed.Access)
	if err != nil {
		return err
	}
	next.Payload = &gamev1.Event_ParticipantJoined{
		ParticipantJoined: &gamev1.ParticipantJoined{
			ParticipantId: typed.ParticipantID,
			Name:          typed.Name,
			Access:        access,
		},
	}
	return nil
}

func encodeParticipantUpdatedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(participant.Updated)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	access, err := protoAccess(typed.Access)
	if err != nil {
		return err
	}
	next.Payload = &gamev1.Event_ParticipantUpdated{
		ParticipantUpdated: &gamev1.ParticipantUpdated{
			ParticipantId: typed.ParticipantID,
			Name:          typed.Name,
			Access:        access,
		},
	}
	return nil
}

func encodeParticipantBoundPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(participant.Bound)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_ParticipantBound{
		ParticipantBound: &gamev1.ParticipantBound{ParticipantId: typed.ParticipantID},
	}
	return nil
}

func encodeParticipantUnboundPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(participant.Unbound)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_ParticipantUnbound{
		ParticipantUnbound: &gamev1.ParticipantUnbound{ParticipantId: typed.ParticipantID},
	}
	return nil
}

func encodeParticipantLeftPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(participant.Left)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_ParticipantLeft{
		ParticipantLeft: &gamev1.ParticipantLeft{ParticipantId: typed.ParticipantID},
	}
	return nil
}

func encodeSceneCreatedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(scene.Created)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_SceneCreated{
		SceneCreated: &gamev1.SceneCreated{
			SceneId:      typed.SceneID,
			SessionId:    typed.SessionID,
			Name:         typed.Name,
			CharacterIds: append([]string{}, typed.CharacterIDs...),
		},
	}
	return nil
}

func encodeSceneActivatedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(scene.Activated)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_SceneActivated{
		SceneActivated: &gamev1.SceneActivated{SceneId: typed.SceneID},
	}
	return nil
}

func encodeSceneEndedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(scene.Ended)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_SceneEnded{
		SceneEnded: &gamev1.SceneEnded{SceneId: typed.SceneID},
	}
	return nil
}

func encodeSceneCastReplacedPayload(next *gamev1.Event, message event.Message) error {
	typed, ok := message.(scene.CastReplaced)
	if !ok {
		return fmt.Errorf("unsupported event payload %T", message)
	}
	next.Payload = &gamev1.Event_SceneCastReplaced{
		SceneCastReplaced: &gamev1.SceneCastReplaced{
			SceneId:      typed.SceneID,
			CharacterIds: append([]string{}, typed.CharacterIDs...),
		},
	}
	return nil
}
