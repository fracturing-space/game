package character

import (
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

const (
	// CommandTypeCreate creates one character in an existing campaign.
	CommandTypeCreate command.Type = "character.create"
	// CommandTypeUpdate updates one existing character.
	CommandTypeUpdate command.Type = "character.update"
	// CommandTypeDelete removes one existing character from active membership.
	CommandTypeDelete command.Type = "character.delete"
	// EventTypeCreated records one character on the campaign timeline.
	EventTypeCreated event.Type = "character.created"
	// EventTypeUpdated records one character metadata replacement.
	EventTypeUpdated event.Type = "character.updated"
	// EventTypeDeleted records one character deletion.
	EventTypeDeleted event.Type = "character.deleted"
)

// Create requests one new character in an existing campaign.
type Create struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

// CommandType returns the stable command identifier.
func (Create) CommandType() command.Type { return CommandTypeCreate }

// Update requests one existing character replacement.
type Update struct {
	CharacterID   string `json:"character_id"`
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

// CommandType returns the stable command identifier.
func (Update) CommandType() command.Type { return CommandTypeUpdate }

// Delete requests one character deletion.
type Delete struct {
	CharacterID string `json:"character_id"`
	Reason      string `json:"reason,omitempty"`
}

// CommandType returns the stable command identifier.
func (Delete) CommandType() command.Type { return CommandTypeDelete }

// Created records one character node on the campaign timeline.
type Created struct {
	CharacterID   string `json:"character_id"`
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

// EventType returns the stable event identifier.
func (Created) EventType() event.Type { return EventTypeCreated }

// Updated records one character replacement.
type Updated struct {
	CharacterID   string `json:"character_id"`
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

// EventType returns the stable event identifier.
func (Updated) EventType() event.Type { return EventTypeUpdated }

// Deleted records one character deletion.
type Deleted struct {
	CharacterID string `json:"character_id"`
}

// EventType returns the stable event identifier.
func (Deleted) EventType() event.Type { return EventTypeDeleted }

// Record is the folded character node stored inside campaign state.
type Record struct {
	ID            string `json:"id"`
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Active        bool   `json:"-"`
}

// ValidateCreate checks the character command invariants.
func ValidateCreate(message Create) error {
	message = normalizeCreate(message)
	if err := canonical.ValidateID(message.ParticipantID, "character participant id"); err != nil {
		return err
	}
	return canonical.ValidateName(message.Name, "character name", canonical.DisplayNameMaxRunes)
}

// ValidateUpdate checks the character.update command invariants.
func ValidateUpdate(message Update) error {
	if err := canonical.ValidateID(message.CharacterID, "character id"); err != nil {
		return err
	}
	return ValidateCreated(Created(message))
}

// ValidateDelete checks the character.delete command invariants.
func ValidateDelete(message Delete) error {
	return canonical.ValidateID(message.CharacterID, "character id")
}

// ValidateCreated checks the character event invariants.
func ValidateCreated(message Created) error {
	message = normalizeCreated(message)
	if err := canonical.ValidateID(message.CharacterID, "character id"); err != nil {
		return err
	}
	return ValidateCreate(Create{
		ParticipantID: message.ParticipantID,
		Name:          message.Name,
	})
}

// ValidateUpdated checks the character.updated event invariants.
func ValidateUpdated(message Updated) error {
	return ValidateUpdate(Update(message))
}

// ValidateDeleted checks the character.deleted event invariants.
func ValidateDeleted(message Deleted) error {
	return ValidateDelete(Delete{
		CharacterID: message.CharacterID,
	})
}

// CreateCommandSpec is the typed character.create contract.
var CreateCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Create]{
	Message:   Create{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeCreate,
	Validate:  ValidateCreate,
})

// UpdateCommandSpec is the typed character.update contract.
var UpdateCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Update]{
	Message:   Update{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeUpdate,
	Validate:  ValidateUpdate,
})

// DeleteCommandSpec is the typed character.delete contract.
var DeleteCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Delete]{
	Message:   Delete{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeDelete,
	Validate:  ValidateDelete,
})

// CreatedEventSpec is the typed character.created contract.
var CreatedEventSpec = event.NewCoreSpec(Created{}, normalizeCreated, ValidateCreated)

// UpdatedEventSpec is the typed character.updated contract.
var UpdatedEventSpec = event.NewCoreSpec(Updated{}, normalizeUpdated, ValidateUpdated)

// DeletedEventSpec is the typed character.deleted contract.
var DeletedEventSpec = event.NewCoreSpec(Deleted{}, normalizeDeleted, ValidateDeleted)

func normalizeCreate(message Create) Create {
	return normalizeCreateBase(message)
}

func normalizeUpdate(message Update) Update {
	return normalizeUpdateBase(message)
}

func normalizeCreated(message Created) Created {
	return normalizeCreatedBase(message)
}

func normalizeUpdated(message Updated) Updated {
	return normalizeUpdatedBase(message)
}
