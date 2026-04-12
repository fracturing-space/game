package participant

import (
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/participant"
)

func requireActiveParticipant(state campaign.State, participantID string) (participant.Record, error) {
	record, ok := state.Participants[participantID]
	if !ok || !record.Active {
		return participant.Record{}, errs.NotFoundf("participant %s not found", participantID)
	}
	return record, nil
}

func authorizeSeatManagement(state campaign.State, act caller.Caller, record participant.Record, requireBoundSeat bool, capability authz.Capability) error {
	// This helper assumes caller-specific invariants such as "owner participants
	// cannot be unbound or removed" are enforced by the calling decider before participant
	// access is evaluated here.
	if record.Access != participant.AccessOwner {
		if owner, ok := campaign.BoundParticipant(state, act.SubjectID); ok && owner.Access == participant.AccessOwner {
			return nil
		}
	}
	if record.SubjectID == "" {
		if requireBoundSeat {
			return authz.Denied(capability, "bound participant access is required")
		}
		return nil
	}
	if record.SubjectID != act.SubjectID {
		return authz.Denied(capability, "participant access is required")
	}
	return nil
}
