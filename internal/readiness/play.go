package readiness

import "strings"

const (
	// RejectionCodePlayReadinessAIAgentRequired indicates AI-GM campaigns require a bound AI agent.
	RejectionCodePlayReadinessAIAgentRequired = "PLAY_READINESS_AI_AGENT_REQUIRED"
	// RejectionCodePlayReadinessPlayerRequired indicates campaigns require at least one bound player participant.
	RejectionCodePlayReadinessPlayerRequired = "PLAY_READINESS_PLAYER_REQUIRED"
	// RejectionCodePlayReadinessPlayerCharacterRequired indicates a bound player participant needs at least one active character.
	RejectionCodePlayReadinessPlayerCharacterRequired = "PLAY_READINESS_PLAYER_CHARACTER_REQUIRED"
)

// ResolutionKind identifies one stable UI/action mapping for a readiness blocker.
type ResolutionKind string

const (
	// ResolutionKindUnspecified indicates the blocker has no direct self-service resolution target.
	ResolutionKindUnspecified ResolutionKind = ""
	// ResolutionKindConfigureAIAgent asks the responsible owner to bind an AI agent.
	ResolutionKindConfigureAIAgent ResolutionKind = "configure_ai_agent"
	// ResolutionKindManageParticipants asks the responsible owner to manage participants.
	ResolutionKindManageParticipants ResolutionKind = "manage_participants"
	// ResolutionKindInvitePlayer asks the responsible owner to invite or bind another player.
	ResolutionKindInvitePlayer ResolutionKind = "invite_player"
	// ResolutionKindCreateCharacter asks the blocked participant to create their own character.
	ResolutionKindCreateCharacter ResolutionKind = "create_character"
)

// Action carries structured responsibility and resolution data for a blocker.
type Action struct {
	ResponsibleParticipantIDs []string
	ResolutionKind            ResolutionKind
	TargetParticipantID       string
}

// Blocker describes one readiness invariant currently preventing entry into play mode.
type Blocker struct {
	Code     string
	Message  string
	Metadata map[string]string
	Action   Action
}

// Report captures all play-readiness blockers for deterministic caller feedback.
type Report struct {
	Blockers []Blocker
}

// Rejection describes why play-readiness evaluation failed.
type Rejection struct {
	Code    string
	Message string
}

// Error implements error so callers can return the rejection directly.
func (r *Rejection) Error() string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Message)
}

// Ready reports whether the campaign has zero play-readiness blockers.
func (r Report) Ready() bool {
	return len(r.Blockers) == 0
}
