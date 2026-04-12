package participant

import (
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
)

const (
	// CommandTypeJoin adds one participant to an existing campaign.
	CommandTypeJoin command.Type = "participant.join"
	// CommandTypeUpdate updates one existing participant.
	CommandTypeUpdate command.Type = "participant.update"
	// CommandTypeBind binds one human participant to a subject.
	CommandTypeBind command.Type = "participant.bind"
	// CommandTypeUnbind clears one human participant binding.
	CommandTypeUnbind command.Type = "participant.unbind"
	// CommandTypeLeave removes one participant from active campaign membership.
	CommandTypeLeave command.Type = "participant.leave"
	// EventTypeJoined records one participant joining the campaign timeline.
	EventTypeJoined event.Type = "participant.joined"
	// EventTypeUpdated records one participant metadata update.
	EventTypeUpdated event.Type = "participant.updated"
	// EventTypeBound records one participant binding.
	EventTypeBound event.Type = "participant.bound"
	// EventTypeUnbound records one participant unbinding.
	EventTypeUnbound event.Type = "participant.unbound"
	// EventTypeLeft records one participant leaving the campaign.
	EventTypeLeft event.Type = "participant.left"
)

// Access declares the participant's campaign access level.
type Access string

const (
	// AccessOwner marks the single campaign owner.
	AccessOwner Access = "OWNER"
	// AccessMember marks a regular campaign member.
	AccessMember Access = "MEMBER"
)

// Join requests one new participant on an existing campaign timeline.
type Join struct {
	Name      string `json:"name"`
	Access    Access `json:"access"`
	SubjectID string `json:"-"`
}

// CommandType returns the stable command identifier.
func (Join) CommandType() command.Type { return CommandTypeJoin }

// Update requests one participant metadata replacement.
type Update struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Access        Access `json:"access"`
}

// CommandType returns the stable command identifier.
func (Update) CommandType() command.Type { return CommandTypeUpdate }

// Bind requests one human participant binding.
type Bind struct {
	ParticipantID string `json:"participant_id"`
	SubjectID     string `json:"-"`
}

// CommandType returns the stable command identifier.
func (Bind) CommandType() command.Type { return CommandTypeBind }

// Unbind requests clearing one human participant binding.
type Unbind struct {
	ParticipantID string `json:"participant_id"`
}

// CommandType returns the stable command identifier.
func (Unbind) CommandType() command.Type { return CommandTypeUnbind }

// Leave requests removing one participant from active membership.
type Leave struct {
	ParticipantID string `json:"participant_id"`
	Reason        string `json:"reason,omitempty"`
}

// CommandType returns the stable command identifier.
func (Leave) CommandType() command.Type { return CommandTypeLeave }

// Joined records one participant node on the campaign timeline.
type Joined struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Access        Access `json:"access"`
	SubjectID     string `json:"-"`
}

// EventType returns the stable event identifier.
func (Joined) EventType() event.Type { return EventTypeJoined }

// Updated records one participant metadata replacement.
type Updated struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Access        Access `json:"access"`
}

// EventType returns the stable event identifier.
func (Updated) EventType() event.Type { return EventTypeUpdated }

// Bound records one participant binding.
type Bound struct {
	ParticipantID string `json:"participant_id"`
	SubjectID     string `json:"-"`
}

// EventType returns the stable event identifier.
func (Bound) EventType() event.Type { return EventTypeBound }

// Unbound records one participant unbinding.
type Unbound struct {
	ParticipantID string `json:"participant_id"`
}

// EventType returns the stable event identifier.
func (Unbound) EventType() event.Type { return EventTypeUnbound }

// Left records one participant removal.
type Left struct {
	ParticipantID string `json:"participant_id"`
}

// EventType returns the stable event identifier.
func (Left) EventType() event.Type { return EventTypeLeft }

// Record is the folded participant node stored inside campaign state.
type Record struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Access    Access `json:"access"`
	SubjectID string `json:"-"`
	Active    bool   `json:"-"`
}

// ValidateJoin checks the participant command invariants.
func ValidateJoin(message Join) error {
	message = normalizeJoin(message)
	if err := canonical.ValidateName(message.Name, "participant name", canonical.DisplayNameMaxRunes); err != nil {
		return err
	}
	return validateIdentity(message.Access, message.SubjectID)
}

// ValidateUpdate checks the participant update command invariants.
func ValidateUpdate(message Update) error {
	message = normalizeUpdate(message)
	if err := canonical.ValidateID(message.ParticipantID, "participant id"); err != nil {
		return err
	}
	return ValidateJoined(Joined{
		ParticipantID: message.ParticipantID,
		Name:          message.Name,
		Access:        message.Access,
	})
}

// ValidateBind checks the bind command invariants.
func ValidateBind(message Bind) error {
	if err := canonical.ValidateID(message.ParticipantID, "participant id"); err != nil {
		return err
	}
	return canonical.ValidateID(message.SubjectID, "participant subject id")
}

// ValidateUnbind checks the unbind command invariants.
func ValidateUnbind(message Unbind) error {
	return canonical.ValidateID(message.ParticipantID, "participant id")
}

// ValidateLeave checks the leave command invariants.
func ValidateLeave(message Leave) error {
	return canonical.ValidateID(message.ParticipantID, "participant id")
}

// ValidateJoined checks the participant event invariants.
func ValidateJoined(message Joined) error {
	message = normalizeJoined(message)
	if err := canonical.ValidateID(message.ParticipantID, "participant id"); err != nil {
		return err
	}
	if err := canonical.ValidateName(message.Name, "participant name", canonical.DisplayNameMaxRunes); err != nil {
		return err
	}
	return validateIdentity(message.Access, message.SubjectID)
}

// ValidateUpdated checks the participant.updated event invariants.
func ValidateUpdated(message Updated) error {
	return ValidateUpdate(Update(message))
}

// ValidateBound checks the participant.bound event invariants.
func ValidateBound(message Bound) error {
	return ValidateBind(Bind(message))
}

// ValidateUnbound checks the participant.unbound event invariants.
func ValidateUnbound(message Unbound) error {
	return ValidateUnbind(Unbind(message))
}

// ValidateLeft checks the participant.left event invariants.
func ValidateLeft(message Left) error {
	return ValidateLeave(Leave{ParticipantID: message.ParticipantID})
}

// Valid reports whether the access value is recognized.
func (a Access) Valid() bool {
	switch a {
	case AccessOwner, AccessMember:
		return true
	default:
		return false
	}
}

// JoinCommandSpec is the typed participant.join contract.
var JoinCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Join]{
	Message:   Join{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeJoin,
	Validate:  ValidateJoin,
})

// UpdateCommandSpec is the typed participant.update contract.
var UpdateCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Update]{
	Message:   Update{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeUpdate,
	Validate:  ValidateUpdate,
})

// BindCommandSpec is the typed participant.bind contract.
var BindCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Bind]{
	Message:   Bind{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeBind,
	Validate:  ValidateBind,
})

// UnbindCommandSpec is the typed participant.unbind contract.
var UnbindCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Unbind]{
	Message:   Unbind{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeUnbind,
	Validate:  ValidateUnbind,
})

// LeaveCommandSpec is the typed participant.leave contract.
var LeaveCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[Leave]{
	Message:   Leave{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeLeave,
	Validate:  ValidateLeave,
})

// JoinedEventSpec is the typed participant.joined contract.
var JoinedEventSpec = event.NewCoreSpec(Joined{}, normalizeJoined, ValidateJoined)

// UpdatedEventSpec is the typed participant.updated contract.
var UpdatedEventSpec = event.NewCoreSpec(Updated{}, normalizeUpdated, ValidateUpdated)

// BoundEventSpec is the typed participant.bound contract.
var BoundEventSpec = event.NewCoreSpec(Bound{}, normalizeBound, ValidateBound)

// UnboundEventSpec is the typed participant.unbound contract.
var UnboundEventSpec = event.NewCoreSpec(Unbound{}, normalizeUnbound, ValidateUnbound)

// LeftEventSpec is the typed participant.left contract.
var LeftEventSpec = event.NewCoreSpec(Left{}, normalizeLeft, ValidateLeft)

func normalizeJoin(message Join) Join {
	message = normalizeJoinBase(message)
	if message.Access == "" {
		message.Access = AccessMember
	}
	return message
}

func validateIdentity(access Access, subjectID string) error {
	if !canonical.IsExact(string(access)) {
		return errs.InvalidArgumentf("participant access must not contain surrounding whitespace")
	}
	if err := canonical.ValidateOptionalID(subjectID, "participant subject id"); err != nil {
		return err
	}
	if !access.Valid() {
		return errs.InvalidArgumentf("participant access is invalid: %s", access)
	}
	if access == AccessOwner && subjectID == "" {
		return errs.InvalidArgumentf("owner participant subject id is required")
	}
	return nil
}
