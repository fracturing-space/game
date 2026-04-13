package campaign

import "github.com/fracturing-space/game/internal/command"

const (
	// CommandTypeCreate creates a new campaign timeline.
	CommandTypeCreate command.Type = "campaign.create"
	// CommandTypeUpdate updates campaign metadata.
	CommandTypeUpdate command.Type = "campaign.update"
	// CommandTypeAIBind binds one AI agent to the campaign.
	CommandTypeAIBind command.Type = "campaign.ai_bind"
	// CommandTypeAIUnbind clears one AI agent binding from the campaign.
	CommandTypeAIUnbind command.Type = "campaign.ai_unbind"
	// CommandTypePlayBegin enters active play for the current session.
	CommandTypePlayBegin command.Type = "campaign.play.begin"
	// CommandTypePlayEnd leaves active play for the current session.
	CommandTypePlayEnd command.Type = "campaign.play.end"
	// CommandTypePlayPause pauses active play for out-of-character coordination.
	CommandTypePlayPause command.Type = "campaign.play.pause"
	// CommandTypePlayResume resumes active play from a paused state.
	CommandTypePlayResume command.Type = "campaign.play.resume"
)

// Create is the root campaign command.
type Create struct {
	Name      string `json:"name"`
	OwnerName string `json:"owner_name"`
}

// CommandType returns the stable command identifier.
func (Create) CommandType() command.Type { return CommandTypeCreate }

// Update requests one campaign metadata replacement.
type Update struct {
	Name string `json:"name"`
}

// CommandType returns the stable command identifier.
func (Update) CommandType() command.Type { return CommandTypeUpdate }

// AIBind requests one campaign-level AI agent binding.
type AIBind struct {
	AIAgentID string `json:"ai_agent_id"`
}

// CommandType returns the stable command identifier.
func (AIBind) CommandType() command.Type { return CommandTypeAIBind }

// AIUnbind requests one cleared campaign-level AI agent binding.
type AIUnbind struct{}

// CommandType returns the stable command identifier.
func (AIUnbind) CommandType() command.Type { return CommandTypeAIUnbind }

// PlayBegin requests entering active play for the current session.
type PlayBegin struct{}

// CommandType returns the stable command identifier.
func (PlayBegin) CommandType() command.Type { return CommandTypePlayBegin }

// PlayEnd requests leaving active play for the current session.
type PlayEnd struct{}

// CommandType returns the stable command identifier.
func (PlayEnd) CommandType() command.Type { return CommandTypePlayEnd }

// PlayPause requests pausing active play.
type PlayPause struct {
	Reason string `json:"reason,omitempty"`
}

// CommandType returns the stable command identifier.
func (PlayPause) CommandType() command.Type { return CommandTypePlayPause }

// PlayResume requests resuming active play from a paused state.
type PlayResume struct {
	Reason string `json:"reason,omitempty"`
}

// CommandType returns the stable command identifier.
func (PlayResume) CommandType() command.Type { return CommandTypePlayResume }
