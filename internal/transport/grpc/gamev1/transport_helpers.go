package gamev1

import (
	"context"
	"fmt"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func callerFromContext(ctx context.Context) (caller.Caller, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return caller.Caller{}, status.Error(codes.Unauthenticated, "subject id is required")
	}
	values := md.Get(subjectIDHeader)
	if len(values) == 0 {
		return caller.Caller{}, status.Error(codes.Unauthenticated, "subject id is required")
	}
	act, err := caller.NewSubject(values[0])
	if err != nil {
		return caller.Caller{}, status.Error(codes.Unauthenticated, "subject id is required")
	}
	return act, nil
}

func resultCampaignID(result service.Result) (string, error) {
	if result.State.ID != "" {
		return result.State.ID, nil
	}
	for _, next := range result.Events {
		if next.CampaignID != "" {
			return next.CampaignID, nil
		}
	}
	for _, next := range result.StoredEvents {
		if next.CampaignID != "" {
			return next.CampaignID, nil
		}
	}
	return "", fmt.Errorf("campaign id is required")
}

func resultParticipantID(result service.Result) (string, error) {
	for _, next := range result.Events {
		joined, err := event.MessageAs[participant.Joined](next)
		if err == nil {
			return joined.ParticipantID, nil
		}
	}
	for _, next := range result.StoredEvents {
		joined, err := event.MessageAs[participant.Joined](next.Envelope)
		if err == nil {
			return joined.ParticipantID, nil
		}
	}
	return "", fmt.Errorf("participant.joined event is required")
}

func resultParticipantEventID(result service.Result) (string, error) {
	for _, next := range result.Events {
		if bound, err := event.MessageAs[participant.Bound](next); err == nil {
			return bound.ParticipantID, nil
		}
		if unbound, err := event.MessageAs[participant.Unbound](next); err == nil {
			return unbound.ParticipantID, nil
		}
		if left, err := event.MessageAs[participant.Left](next); err == nil {
			return left.ParticipantID, nil
		}
		if updated, err := event.MessageAs[participant.Updated](next); err == nil {
			return updated.ParticipantID, nil
		}
	}
	for _, next := range result.StoredEvents {
		if bound, err := event.MessageAs[participant.Bound](next.Envelope); err == nil {
			return bound.ParticipantID, nil
		}
		if unbound, err := event.MessageAs[participant.Unbound](next.Envelope); err == nil {
			return unbound.ParticipantID, nil
		}
		if left, err := event.MessageAs[participant.Left](next.Envelope); err == nil {
			return left.ParticipantID, nil
		}
		if updated, err := event.MessageAs[participant.Updated](next.Envelope); err == nil {
			return updated.ParticipantID, nil
		}
	}
	return "", fmt.Errorf("participant lifecycle event is required")
}

func resultCharacterID(result service.Result) (string, error) {
	for _, next := range result.Events {
		created, err := event.MessageAs[character.Created](next)
		if err == nil {
			return created.CharacterID, nil
		}
	}
	for _, next := range result.StoredEvents {
		created, err := event.MessageAs[character.Created](next.Envelope)
		if err == nil {
			return created.CharacterID, nil
		}
	}
	return "", fmt.Errorf("character.created event is required")
}

func resultCharacterEventID(result service.Result) (string, error) {
	for _, next := range result.Events {
		if updated, err := event.MessageAs[character.Updated](next); err == nil {
			return updated.CharacterID, nil
		}
		if deleted, err := event.MessageAs[character.Deleted](next); err == nil {
			return deleted.CharacterID, nil
		}
	}
	for _, next := range result.StoredEvents {
		if updated, err := event.MessageAs[character.Updated](next.Envelope); err == nil {
			return updated.CharacterID, nil
		}
		if deleted, err := event.MessageAs[character.Deleted](next.Envelope); err == nil {
			return deleted.CharacterID, nil
		}
	}
	return "", fmt.Errorf("character lifecycle event is required")
}

func resultSceneID(result service.Result) (string, error) {
	for _, next := range result.Events {
		created, err := event.MessageAs[scene.Created](next)
		if err == nil {
			return created.SceneID, nil
		}
	}
	for _, next := range result.StoredEvents {
		created, err := event.MessageAs[scene.Created](next.Envelope)
		if err == nil {
			return created.SceneID, nil
		}
	}
	return "", fmt.Errorf("scene.created event is required")
}

func resultSceneEventID(result service.Result) (string, error) {
	for _, next := range result.Events {
		if activated, err := event.MessageAs[scene.Activated](next); err == nil {
			return activated.SceneID, nil
		}
		if ended, err := event.MessageAs[scene.Ended](next); err == nil {
			return ended.SceneID, nil
		}
		if replaced, err := event.MessageAs[scene.CastReplaced](next); err == nil {
			return replaced.SceneID, nil
		}
	}
	for _, next := range result.StoredEvents {
		if activated, err := event.MessageAs[scene.Activated](next.Envelope); err == nil {
			return activated.SceneID, nil
		}
		if ended, err := event.MessageAs[scene.Ended](next.Envelope); err == nil {
			return ended.SceneID, nil
		}
		if replaced, err := event.MessageAs[scene.CastReplaced](next.Envelope); err == nil {
			return replaced.SceneID, nil
		}
	}
	return "", fmt.Errorf("scene lifecycle event is required")
}

func resultScene(result service.Result, sceneID string) (*gamev1.Scene, error) {
	if err := canonical.ValidateExact(sceneID, "scene id", fmt.Errorf); err != nil {
		return nil, err
	}
	for _, next := range result.State.Scenes {
		if next.ID == sceneID {
			return protoScene(next), nil
		}
	}
	return nil, fmt.Errorf("scene %s is required in result state", sceneID)
}

func resultSession(result service.Result) (*gamev1.Session, error) {
	for _, next := range result.Events {
		if started, err := event.MessageAs[session.Started](next); err == nil {
			return protoSession(started.SessionID, started.Name, session.StatusActive, started.CharacterControllers)
		}
		if ended, err := event.MessageAs[session.Ended](next); err == nil {
			return protoSession(ended.SessionID, ended.Name, session.StatusEnded, ended.CharacterControllers)
		}
	}
	for _, next := range result.StoredEvents {
		if started, err := event.MessageAs[session.Started](next.Envelope); err == nil {
			return protoSession(started.SessionID, started.Name, session.StatusActive, started.CharacterControllers)
		}
		if ended, err := event.MessageAs[session.Ended](next.Envelope); err == nil {
			return protoSession(ended.SessionID, ended.Name, session.StatusEnded, ended.CharacterControllers)
		}
	}
	return nil, fmt.Errorf("session lifecycle event is required")
}
