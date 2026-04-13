package campaign

import "github.com/fracturing-space/game/internal/event"

const (
	// EventTypeCreated records a newly created campaign.
	EventTypeCreated event.Type = "campaign.created"
	// EventTypeUpdated records updated campaign metadata.
	EventTypeUpdated event.Type = "campaign.updated"
	// EventTypeAIBound records one AI agent binding for the campaign.
	EventTypeAIBound event.Type = "campaign.ai_bound"
	// EventTypeAIUnbound records one cleared AI agent binding for the campaign.
	EventTypeAIUnbound event.Type = "campaign.ai_unbound"
	// EventTypePlayBegan records entry into active play.
	EventTypePlayBegan event.Type = "campaign.play.began"
	// EventTypePlayPaused records one pause from active play.
	EventTypePlayPaused event.Type = "campaign.play.paused"
	// EventTypePlayResumed records one resume from a paused state.
	EventTypePlayResumed event.Type = "campaign.play.resumed"
	// EventTypePlayEnded records one transition out of active play.
	EventTypePlayEnded event.Type = "campaign.play.ended"
)

// Created records campaign creation.
type Created struct {
	Name string `json:"name"`
}

// EventType returns the stable event identifier.
func (Created) EventType() event.Type { return EventTypeCreated }

// Updated records campaign metadata replacement.
type Updated struct {
	Name string `json:"name"`
}

// EventType returns the stable event identifier.
func (Updated) EventType() event.Type { return EventTypeUpdated }

// AIBound records one campaign-level AI agent binding.
type AIBound struct {
	AIAgentID string `json:"ai_agent_id"`
}

// EventType returns the stable event identifier.
func (AIBound) EventType() event.Type { return EventTypeAIBound }

// AIUnbound records one cleared campaign-level AI agent binding.
type AIUnbound struct{}

// EventType returns the stable event identifier.
func (AIUnbound) EventType() event.Type { return EventTypeAIUnbound }

// PlayBegan records entry into active play.
type PlayBegan struct {
	SessionID string `json:"session_id"`
	SceneID   string `json:"scene_id"`
}

// EventType returns the stable event identifier.
func (PlayBegan) EventType() event.Type { return EventTypePlayBegan }

// PlayPaused records one pause from active play.
type PlayPaused struct {
	SessionID string `json:"session_id"`
	SceneID   string `json:"scene_id"`
	Reason    string `json:"reason,omitempty"`
}

// EventType returns the stable event identifier.
func (PlayPaused) EventType() event.Type { return EventTypePlayPaused }

// PlayResumed records one resume from a paused state.
type PlayResumed struct {
	SessionID string `json:"session_id"`
	SceneID   string `json:"scene_id"`
	Reason    string `json:"reason,omitempty"`
}

// EventType returns the stable event identifier.
func (PlayResumed) EventType() event.Type { return EventTypePlayResumed }

// PlayEnded records one transition out of active play.
type PlayEnded struct {
	SessionID string `json:"session_id"`
	SceneID   string `json:"scene_id,omitempty"`
}

// EventType returns the stable event identifier.
func (PlayEnded) EventType() event.Type { return EventTypePlayEnded }
