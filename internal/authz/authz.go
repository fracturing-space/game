package authz

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/participant"
)

// Capability labels one authorization surface.
type Capability string

const (
	CapabilityAuthenticated Capability = "authenticated"

	CapabilityCreateCampaign  Capability = "create_campaign"
	CapabilityUpdateCampaign  Capability = "update_campaign"
	CapabilityManageAIBinding Capability = "manage_ai_binding"
	CapabilityReadCampaign    Capability = "read_campaign"

	CapabilityCreateParticipant Capability = "create_participant"
	CapabilityUpdateParticipant Capability = "update_participant"
	CapabilityBindParticipant   Capability = "bind_participant"
	CapabilityUnbindParticipant Capability = "unbind_participant"
	CapabilityDeleteParticipant Capability = "delete_participant"

	CapabilityCreateCharacter Capability = "create_character"
	CapabilityUpdateCharacter Capability = "update_character"
	CapabilityDeleteCharacter Capability = "delete_character"

	CapabilityStartSession Capability = "start_session"
	CapabilityEndSession   Capability = "end_session"

	CapabilityCreateScene      Capability = "create_scene"
	CapabilityActivateScene    Capability = "activate_scene"
	CapabilityEndScene         Capability = "end_scene"
	CapabilityReplaceSceneCast Capability = "replace_scene_cast"
	CapabilityPlanCommands     Capability = "plan_commands"

	CapabilityBeginPlay  Capability = "begin_play"
	CapabilityPausePlay  Capability = "pause_play"
	CapabilityResumePlay Capability = "resume_play"
	CapabilityEndPlay    Capability = "end_play"
)

// DeniedError marks one authorization failure.
type DeniedError struct {
	Capability Capability
	Reason     string
}

func (e *DeniedError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Reason) == "" {
		return fmt.Sprintf("%s denied", e.Capability)
	}
	return fmt.Sprintf("%s denied: %s", e.Capability, e.Reason)
}

// IsDenied reports whether the error is an authorization denial.
func IsDenied(err error) bool {
	var denied *DeniedError
	return errors.As(err, &denied)
}

// Denied constructs one authorization denial with stable capability metadata.
func Denied(capability Capability, reason string) error {
	return &DeniedError{
		Capability: capability,
		Reason:     reason,
	}
}

// RequireCreateCampaign authorizes campaign creation.
func RequireCreateCampaign(call caller.Caller) error {
	return requireAuthenticated(call, CapabilityCreateCampaign)
}

// RequireCaller validates that the caller identity is present.
func RequireCaller(call caller.Caller) error {
	return requireAuthenticated(call, CapabilityAuthenticated)
}

// RequireUpdateCampaign authorizes campaign metadata updates for the owner.
func RequireUpdateCampaign(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityUpdateCampaign)
}

// RequireManageAIBinding authorizes campaign AI binding changes for the owner.
func RequireManageAIBinding(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityManageAIBinding)
}

// RequireReadCampaign authorizes campaign reads for bound participants only.
func RequireReadCampaign(call caller.Caller, state campaign.State) error {
	if err := requireAuthenticated(call, CapabilityReadCampaign); err != nil {
		return err
	}
	if _, ok := campaign.CallerParticipant(state, call); ok {
		return nil
	}
	return Denied(CapabilityReadCampaign, "campaign participant binding is required")
}

// RequireBeginPlay authorizes explicit play entry for the owner.
func RequireBeginPlay(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityBeginPlay)
}

// RequirePausePlay authorizes explicit play pause for the owner.
func RequirePausePlay(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityPausePlay)
}

// RequireResumePlay authorizes explicit play resume for the owner.
func RequireResumePlay(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityResumePlay)
}

// RequireEndPlay authorizes explicit play exit for the owner.
func RequireEndPlay(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityEndPlay)
}

// RequireCreateParticipant authorizes participant creation for the owner.
func RequireCreateParticipant(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityCreateParticipant)
}

// RequireUpdateParticipant authorizes participant updates for the owner.
func RequireUpdateParticipant(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityUpdateParticipant)
}

// RequireBindParticipant authorizes authenticated bind attempts.
func RequireBindParticipant(call caller.Caller) error {
	return requireAuthenticated(call, CapabilityBindParticipant)
}

// RequireUnbindParticipant authorizes authenticated unbind attempts.
func RequireUnbindParticipant(call caller.Caller) error {
	return requireAuthenticated(call, CapabilityUnbindParticipant)
}

// RequireDeleteParticipant authorizes participant deletion for the owner.
func RequireDeleteParticipant(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityDeleteParticipant)
}

// RequireCreateCharacter authorizes authenticated character creation attempts.
func RequireCreateCharacter(call caller.Caller) error {
	return requireAuthenticated(call, CapabilityCreateCharacter)
}

// RequireUpdateCharacter authorizes authenticated character update attempts.
func RequireUpdateCharacter(call caller.Caller) error {
	return requireAuthenticated(call, CapabilityUpdateCharacter)
}

// RequireDeleteCharacter authorizes authenticated character deletion attempts.
func RequireDeleteCharacter(call caller.Caller) error {
	return requireAuthenticated(call, CapabilityDeleteCharacter)
}

// RequireStartSession authorizes session start for the owner.
func RequireStartSession(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityStartSession)
}

// RequireEndSession authorizes session end for the owner.
func RequireEndSession(call caller.Caller, state campaign.State) error {
	return requireOwnerAccess(call, state, CapabilityEndSession)
}

// RequireCreateScene authorizes scene creation for the bound AI agent caller.
func RequireCreateScene(call caller.Caller, state campaign.State) error {
	return requireAIGMAccess(call, state, CapabilityCreateScene)
}

// RequireActivateScene authorizes scene activation for the bound AI agent caller.
func RequireActivateScene(call caller.Caller, state campaign.State) error {
	return requireAIGMAccess(call, state, CapabilityActivateScene)
}

// RequireEndScene authorizes scene end for the bound AI agent caller.
func RequireEndScene(call caller.Caller, state campaign.State) error {
	return requireAIGMAccess(call, state, CapabilityEndScene)
}

// RequireReplaceSceneCast authorizes scene cast replacement for the bound AI agent caller.
func RequireReplaceSceneCast(call caller.Caller, state campaign.State) error {
	return requireAIGMAccess(call, state, CapabilityReplaceSceneCast)
}

// RequirePlanCommands authorizes command planning operations for the bound AI agent caller.
func RequirePlanCommands(call caller.Caller, state campaign.State) error {
	return requireAIGMAccess(call, state, CapabilityPlanCommands)
}

func requireOwnerAccess(call caller.Caller, state campaign.State, capability Capability) error {
	if err := requireAuthenticated(call, capability); err != nil {
		return err
	}
	record, ok := campaign.CallerParticipant(state, call)
	if !ok {
		return Denied(capability, "owner participant binding is required")
	}
	if record.Access != participant.AccessOwner {
		return Denied(capability, "owner access is required")
	}
	return nil
}

func requireAIGMAccess(call caller.Caller, state campaign.State, capability Capability) error {
	if err := requireAuthenticated(call, capability); err != nil {
		return err
	}
	if state.AIAgentID == "" {
		return Denied(capability, "campaign ai binding is required")
	}
	if call.AIAgentID == "" {
		return Denied(capability, "ai agent caller is required")
	}
	if !campaign.CallerMatchesBoundAIAgent(state, call) {
		return Denied(capability, "campaign ai binding does not match caller")
	}
	return nil
}

func requireAuthenticated(call caller.Caller, capability Capability) error {
	if call.Valid() {
		return nil
	}
	return Denied(capability, "caller identity is required")
}
