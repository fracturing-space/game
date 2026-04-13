package campaign

import (
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/participant"
)

// ValidatePlayStateTransition reports whether one play-state transition is allowed.
func ValidatePlayStateTransition(from PlayState, to PlayState) error {
	if !from.Valid() {
		return errs.FailedPreconditionf("current campaign play state is invalid: %s", from)
	}
	if !to.Valid() {
		return errs.InvalidArgumentf("target campaign play state is invalid: %s", to)
	}
	switch {
	case from == PlayStateSetup && to == PlayStateActive:
		return nil
	case from == PlayStateActive && to == PlayStatePaused:
		return nil
	case from == PlayStatePaused && to == PlayStateActive:
		return nil
	case from == PlayStateActive && to == PlayStateSetup:
		return nil
	case from == PlayStatePaused && to == PlayStateSetup:
		return nil
	default:
		return errs.FailedPreconditionf("campaign play state transition is invalid: %s -> %s", from, to)
	}
}

// ValidateCreate checks the create command invariants.
func ValidateCreate(message Create) error {
	message = normalizeCreate(message)
	if err := canonical.ValidateName(message.Name, "campaign name", canonical.DisplayNameMaxRunes); err != nil {
		return err
	}
	if err := canonical.ValidateName(message.OwnerName, "owner name", canonical.DisplayNameMaxRunes); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate checks the update command invariants.
func ValidateUpdate(message Update) error {
	message = normalizeUpdate(message)
	return canonical.ValidateName(message.Name, "campaign name", canonical.DisplayNameMaxRunes)
}

// ValidateAIBind checks the AI-bind command invariants.
func ValidateAIBind(message AIBind) error {
	return canonical.ValidateID(message.AIAgentID, "ai agent id")
}

// ValidateCreated checks the created event invariants.
func ValidateCreated(message Created) error {
	message = normalizeCreated(message)
	return canonical.ValidateName(message.Name, "campaign name", canonical.DisplayNameMaxRunes)
}

// ValidateUpdated checks the updated event invariants.
func ValidateUpdated(message Updated) error {
	return ValidateUpdate(Update(message))
}

// ValidateAIBound checks the AI-bound event invariants.
func ValidateAIBound(message AIBound) error {
	return canonical.ValidateID(message.AIAgentID, "ai agent id")
}

// ValidatePlayBegan checks the play-began event invariants.
func ValidatePlayBegan(message PlayBegan) error {
	if err := canonical.ValidateID(message.SessionID, "session id"); err != nil {
		return err
	}
	return canonical.ValidateOptionalID(message.SceneID, "scene id")
}

// ValidatePlayPaused checks the play-paused event invariants.
func ValidatePlayPaused(message PlayPaused) error {
	if err := canonical.ValidateID(message.SessionID, "session id"); err != nil {
		return err
	}
	return canonical.ValidateID(message.SceneID, "scene id")
}

// ValidatePlayResumed checks the play-resumed event invariants.
func ValidatePlayResumed(message PlayResumed) error {
	if err := canonical.ValidateID(message.SessionID, "session id"); err != nil {
		return err
	}
	return canonical.ValidateID(message.SceneID, "scene id")
}

// ValidatePlayEnded checks the play-ended event invariants.
func ValidatePlayEnded(message PlayEnded) error {
	if err := canonical.ValidateID(message.SessionID, "session id"); err != nil {
		return err
	}
	return canonical.ValidateOptionalID(message.SceneID, "scene id")
}

// HasBoundSubject reports whether the supplied non-empty subject is already
// bound to an active participant in the campaign.
func HasBoundSubject(state State, subjectID string) bool {
	if subjectID == "" {
		return false
	}
	for _, record := range state.Participants {
		if !record.Active {
			continue
		}
		if record.SubjectID == subjectID {
			return true
		}
	}
	return false
}

// BoundParticipant returns the active participant bound to one non-empty subject id.
func BoundParticipant(state State, subjectID string) (participant.Record, bool) {
	if subjectID == "" {
		return participant.Record{}, false
	}
	for _, record := range state.Participants {
		if !record.Active {
			continue
		}
		if record.SubjectID == subjectID {
			return record, true
		}
	}
	return participant.Record{}, false
}

// HasBoundAIAgent reports whether the campaign has a non-empty AI agent binding.
func HasBoundAIAgent(state State) bool {
	return state.AIAgentID != ""
}
