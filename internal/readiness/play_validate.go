package readiness

import (
	"fmt"

	"github.com/fracturing-space/game/internal/canonical"
)

// ValidateReport checks the public readiness-report invariants.
func ValidateReport(report Report) error {
	for _, blocker := range report.Blockers {
		if blocker.Code == "" {
			return fmt.Errorf("play readiness blocker code is required")
		}
		if blocker.Message == "" {
			return fmt.Errorf("play readiness blocker message is required")
		}
		if err := validateAction(blocker.Action); err != nil {
			return fmt.Errorf("play readiness blocker %s action is invalid: %w", blocker.Code, err)
		}
	}
	return nil
}

func validateAction(action Action) error {
	switch action.ResolutionKind {
	case ResolutionKindUnspecified,
		ResolutionKindConfigureAIAgent,
		ResolutionKindManageParticipants,
		ResolutionKindInvitePlayer,
		ResolutionKindCreateCharacter:
	default:
		return fmt.Errorf("resolution kind is invalid: %s", action.ResolutionKind)
	}

	normalized := normalizeIDs(action.ResponsibleParticipantIDs)
	if len(action.ResponsibleParticipantIDs) != len(normalized) {
		return fmt.Errorf("responsible participant ids must be canonical, non-empty, and unique")
	}
	for idx, id := range normalized {
		if action.ResponsibleParticipantIDs[idx] != id {
			return fmt.Errorf("responsible participant ids must be normalized")
		}
		if !canonical.IsExact(id) {
			return fmt.Errorf("responsible participant ids must be canonical")
		}
	}

	targetParticipantID := action.TargetParticipantID
	if targetParticipantID != "" && !canonical.IsExact(targetParticipantID) {
		return fmt.Errorf("target participant id must be canonical")
	}

	switch action.ResolutionKind {
	case ResolutionKindManageParticipants, ResolutionKindInvitePlayer, ResolutionKindUnspecified:
		if targetParticipantID != "" {
			return fmt.Errorf("target participant id is not allowed for resolution kind %s", action.ResolutionKind)
		}
	case ResolutionKindCreateCharacter:
		if targetParticipantID == "" {
			return fmt.Errorf("target participant id is required for resolution kind %s", action.ResolutionKind)
		}
	}
	return nil
}
