package session

import (
	"slices"
	"strings"

	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
)

const (
	// CommandTypeStart starts one active session for a campaign.
	CommandTypeStart command.Type = "session.start"
	// CommandTypeEnd ends the current active session for a campaign.
	CommandTypeEnd command.Type = "session.end"
	// EventTypeStarted records one started session on the campaign timeline.
	EventTypeStarted event.Type = "session.started"
	// EventTypeEnded records one ended session on the campaign timeline.
	EventTypeEnded event.Type = "session.ended"
)

// Status declares the lifecycle state for one session summary.
type Status string

const (
	// StatusActive marks one active session.
	StatusActive Status = "ACTIVE"
	// StatusEnded marks one ended session.
	StatusEnded Status = "ENDED"
)

// CharacterControllerAssignment stores one session-scoped controller override.
type CharacterControllerAssignment struct {
	CharacterID   string `json:"character_id"`
	ParticipantID string `json:"participant_id"`
}

// Start requests one new session for an existing campaign.
type Start struct {
	Name                 string                          `json:"name"`
	CharacterControllers []CharacterControllerAssignment `json:"character_controllers"`
}

// CommandType returns the stable command identifier.
func (Start) CommandType() command.Type { return CommandTypeStart }

// End requests ending the current active session.
type End struct{}

// CommandType returns the stable command identifier.
func (End) CommandType() command.Type { return CommandTypeEnd }

// Started records one active session summary.
type Started struct {
	SessionID            string                          `json:"session_id"`
	Name                 string                          `json:"name"`
	CharacterControllers []CharacterControllerAssignment `json:"character_controllers"`
}

// EventType returns the stable event identifier.
func (Started) EventType() event.Type { return EventTypeStarted }

// Ended records one ended session summary.
type Ended struct {
	SessionID            string                          `json:"session_id"`
	Name                 string                          `json:"name"`
	CharacterControllers []CharacterControllerAssignment `json:"character_controllers"`
}

// EventType returns the stable event identifier.
func (Ended) EventType() event.Type { return EventTypeEnded }

// Record is the folded session summary stored in campaign state.
type Record struct {
	ID                   string                          `json:"id"`
	Name                 string                          `json:"name"`
	Status               Status                          `json:"status"`
	CharacterControllers []CharacterControllerAssignment `json:"character_controllers"`
}

// Valid reports whether the session status value is recognized.
func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusEnded:
		return true
	default:
		return false
	}
}

// CloneAssignments returns a normalized copy sorted by character id.
func CloneAssignments(input []CharacterControllerAssignment) []CharacterControllerAssignment {
	if len(input) == 0 {
		return nil
	}
	cloned := make([]CharacterControllerAssignment, 0, len(input))
	for _, next := range input {
		cloned = append(cloned, CharacterControllerAssignment{
			CharacterID:   next.CharacterID,
			ParticipantID: next.ParticipantID,
		})
	}
	slices.SortFunc(cloned, func(a, b CharacterControllerAssignment) int {
		return strings.Compare(a.CharacterID, b.CharacterID)
	})
	return cloned
}

// CloneRecord returns a deep copy of one session record.
func CloneRecord(input *Record) *Record {
	if input == nil {
		return nil
	}
	return &Record{
		ID:                   input.ID,
		Name:                 input.Name,
		Status:               input.Status,
		CharacterControllers: CloneAssignments(input.CharacterControllers),
	}
}

// ValidateStart checks the session.start command invariants.
func ValidateStart(message Start) error {
	seen := make(map[string]struct{}, len(message.CharacterControllers))
	for _, next := range message.CharacterControllers {
		characterID := next.CharacterID
		if err := canonical.ValidateID(characterID, "session character controller character id"); err != nil {
			return err
		}
		if err := canonical.ValidateID(next.ParticipantID, "session character controller participant id"); err != nil {
			return err
		}
		if _, ok := seen[characterID]; ok {
			return errs.InvalidArgumentf("session character controllers must be unique by character")
		}
		seen[characterID] = struct{}{}
	}
	return nil
}

// ValidateStarted checks the session.started event invariants.
func ValidateStarted(message Started) error {
	if err := canonical.ValidateID(message.SessionID, "session id"); err != nil {
		return err
	}
	if message.Name == "" {
		return errs.InvalidArgumentf("session name is required")
	}
	return validateAssignments(message.CharacterControllers)
}

// ValidateEnded checks the session.ended event invariants.
func ValidateEnded(message Ended) error {
	if err := canonical.ValidateID(message.SessionID, "session id"); err != nil {
		return err
	}
	if message.Name == "" {
		return errs.InvalidArgumentf("session name is required")
	}
	return validateAssignments(message.CharacterControllers)
}

func validateAssignments(input []CharacterControllerAssignment) error {
	seen := make(map[string]struct{}, len(input))
	for _, next := range input {
		characterID := next.CharacterID
		if err := canonical.ValidateID(characterID, "session character controller character id"); err != nil {
			return err
		}
		participantID := next.ParticipantID
		if err := canonical.ValidateID(participantID, "session character controller participant id"); err != nil {
			return err
		}
		if _, ok := seen[characterID]; ok {
			return errs.InvalidArgumentf("session character controllers must be unique by character")
		}
		seen[characterID] = struct{}{}
	}
	return nil
}

// StartCommandSpec is the typed session.start contract.
var StartCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Start]{
	Message:   Start{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeStart,
	Validate:  ValidateStart,
})

// EndCommandSpec is the typed session.end contract.
var EndCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[End]{
	Message: End{},
	Scope:   command.ScopeCampaign,
})

// StartedEventSpec is the typed session.started contract.
var StartedEventSpec = event.NewCoreSpec(Started{}, normalizeStarted, ValidateStarted)

// EndedEventSpec is the typed session.ended contract.
var EndedEventSpec = event.NewCoreSpec(Ended{}, normalizeEnded, ValidateEnded)
