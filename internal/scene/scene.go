package scene

import (
	"slices"

	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

const (
	// CommandTypeCreate creates one scene for the active session.
	CommandTypeCreate command.Type = "scene.create"
	// CommandTypeActivate marks one existing scene active.
	CommandTypeActivate command.Type = "scene.activate"
	// CommandTypeEnd closes one scene.
	CommandTypeEnd command.Type = "scene.end"
	// CommandTypeReplaceCast replaces one scene cast.
	CommandTypeReplaceCast command.Type = "scene.cast.replace"
	// EventTypeCreated records one created scene.
	EventTypeCreated event.Type = "scene.created"
	// EventTypeActivated records one scene activation.
	EventTypeActivated event.Type = "scene.activated"
	// EventTypeEnded records one scene end.
	EventTypeEnded event.Type = "scene.ended"
	// EventTypeCastReplaced records one scene cast replacement.
	EventTypeCastReplaced event.Type = "scene.cast.replaced"
)

// Create requests one new scene in the active session.
type Create struct {
	Name         string   `json:"name"`
	CharacterIDs []string `json:"character_ids,omitempty"`
}

// CommandType returns the stable command identifier.
func (Create) CommandType() command.Type { return CommandTypeCreate }

// Activate requests marking one existing scene active.
type Activate struct {
	SceneID string `json:"scene_id"`
}

// CommandType returns the stable command identifier.
func (Activate) CommandType() command.Type { return CommandTypeActivate }

// End requests closing one scene.
type End struct {
	SceneID string `json:"scene_id"`
}

// CommandType returns the stable command identifier.
func (End) CommandType() command.Type { return CommandTypeEnd }

// ReplaceCast requests replacing one scene cast.
type ReplaceCast struct {
	SceneID      string   `json:"scene_id"`
	CharacterIDs []string `json:"character_ids"`
}

// CommandType returns the stable command identifier.
func (ReplaceCast) CommandType() command.Type { return CommandTypeReplaceCast }

// Created records one created scene.
type Created struct {
	SceneID      string   `json:"scene_id"`
	SessionID    string   `json:"session_id"`
	Name         string   `json:"name"`
	CharacterIDs []string `json:"character_ids,omitempty"`
}

// EventType returns the stable event identifier.
func (Created) EventType() event.Type { return EventTypeCreated }

// Activated records one scene activation.
type Activated struct {
	SceneID string `json:"scene_id"`
}

// EventType returns the stable event identifier.
func (Activated) EventType() event.Type { return EventTypeActivated }

// Ended records one scene end.
type Ended struct {
	SceneID string `json:"scene_id"`
}

// EventType returns the stable event identifier.
func (Ended) EventType() event.Type { return EventTypeEnded }

// CastReplaced records one scene cast replacement.
type CastReplaced struct {
	SceneID      string   `json:"scene_id"`
	CharacterIDs []string `json:"character_ids"`
}

// EventType returns the stable event identifier.
func (CastReplaced) EventType() event.Type { return EventTypeCastReplaced }

// Record stores one replayed scene.
type Record struct {
	ID           string   `json:"id"`
	SessionID    string   `json:"session_id"`
	Name         string   `json:"name"`
	Active       bool     `json:"active"`
	Ended        bool     `json:"ended"`
	CharacterIDs []string `json:"character_ids"`
}

// CloneCharacterIDs returns one deterministic copy of scene character ids.
func CloneCharacterIDs(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	cloned := append([]string(nil), input...)
	slices.Sort(cloned)
	return slices.Compact(cloned)
}

// ValidateCreate checks scene.create invariants.
func ValidateCreate(message Create) error {
	message = normalizeCreate(message)
	if err := canonical.ValidateName(message.Name, "scene name", canonical.DisplayNameMaxRunes); err != nil {
		return err
	}
	for _, characterID := range message.CharacterIDs {
		if err := canonical.ValidateID(characterID, "scene character id"); err != nil {
			return err
		}
	}
	return nil
}

// ValidateActivate checks scene.activate invariants.
func ValidateActivate(message Activate) error {
	return canonical.ValidateID(message.SceneID, "scene id")
}

// ValidateEnd checks scene.end invariants.
func ValidateEnd(message End) error {
	return canonical.ValidateID(message.SceneID, "scene id")
}

// ValidateReplaceCast checks scene.cast.replace invariants.
func ValidateReplaceCast(message ReplaceCast) error {
	if err := canonical.ValidateID(message.SceneID, "scene id"); err != nil {
		return err
	}
	for _, characterID := range message.CharacterIDs {
		if err := canonical.ValidateID(characterID, "scene character id"); err != nil {
			return err
		}
	}
	return nil
}

// ValidateCreated checks scene.created invariants.
func ValidateCreated(message Created) error {
	message = normalizeCreated(message)
	if err := canonical.ValidateID(message.SceneID, "scene id"); err != nil {
		return err
	}
	if err := canonical.ValidateID(message.SessionID, "scene session id"); err != nil {
		return err
	}
	if err := canonical.ValidateName(message.Name, "scene name", canonical.DisplayNameMaxRunes); err != nil {
		return err
	}
	for _, characterID := range message.CharacterIDs {
		if err := canonical.ValidateID(characterID, "scene character id"); err != nil {
			return err
		}
	}
	return nil
}

// ValidateActivated checks scene.activated invariants.
func ValidateActivated(message Activated) error { return ValidateActivate(Activate(message)) }

// ValidateEnded checks scene.ended invariants.
func ValidateEnded(message Ended) error { return ValidateEnd(End(message)) }

// ValidateCastReplaced checks scene.cast.replaced invariants.
func ValidateCastReplaced(message CastReplaced) error {
	return ValidateReplaceCast(ReplaceCast(message))
}

// CreateCommandSpec is the typed scene.create contract.
var CreateCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Create]{
	Message:   Create{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeCreate,
	Validate:  ValidateCreate,
})

// ActivateCommandSpec is the typed scene.activate contract.
var ActivateCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Activate]{
	Message:   Activate{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeActivate,
	Validate:  ValidateActivate,
})

// EndCommandSpec is the typed scene.end contract.
var EndCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[End]{
	Message:   End{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeEnd,
	Validate:  ValidateEnd,
})

// ReplaceCastCommandSpec is the typed scene.cast.replace contract.
var ReplaceCastCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[ReplaceCast]{
	Message:   ReplaceCast{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeReplaceCast,
	Validate:  ValidateReplaceCast,
})

// CreatedEventSpec is the typed scene.created contract.
var CreatedEventSpec = event.NewCoreSpec(Created{}, normalizeCreated, ValidateCreated)

// ActivatedEventSpec is the typed scene.activated contract.
var ActivatedEventSpec = event.NewCoreSpec(Activated{}, normalizeActivated, ValidateActivated)

// EndedEventSpec is the typed scene.ended contract.
var EndedEventSpec = event.NewCoreSpec(Ended{}, normalizeEnded, ValidateEnded)

// CastReplacedEventSpec is the typed scene.cast.replaced contract.
var CastReplacedEventSpec = event.NewCoreSpec(CastReplaced{}, normalizeCastReplacedEvent, ValidateCastReplaced)
