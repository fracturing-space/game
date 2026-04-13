package campaign

import (
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/participant"
)

// CallerParticipant resolves one active campaign participant from a subject caller.
func CallerParticipant(state State, call caller.Caller) (participant.Record, bool) {
	if call.SubjectID == "" {
		return participant.Record{}, false
	}
	return BoundParticipant(state, call.SubjectID)
}

// CallerMatchesBoundAIAgent reports whether the caller is the campaign's bound AI GM.
func CallerMatchesBoundAIAgent(state State, call caller.Caller) bool {
	return call.AIAgentID != "" && call.AIAgentID == state.AIAgentID
}
