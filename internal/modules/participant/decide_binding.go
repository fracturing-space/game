package participant

import (
	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

func decideBind(state campaign.State, act caller.Caller, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[participant.Bind](envelope)
	if err != nil {
		return nil, err
	}
	record, err := requireActiveParticipant(state, message.ParticipantID)
	if err != nil {
		return nil, err
	}
	if err := authorizeSeatManagement(state, act, record, false, authz.CapabilityBindParticipant); err != nil {
		return nil, err
	}
	if record.SubjectID != "" {
		return nil, errs.Conflictf("participant %s is already bound", record.ID)
	}
	if campaign.HasBoundSubject(state, act.SubjectID) {
		return nil, errs.Conflictf("subject is already bound in campaign")
	}
	bound, err := event.NewEnvelope(
		participant.BoundEventSpec,
		envelope.CampaignID,
		participant.Bound{
			ParticipantID: record.ID,
			SubjectID:     act.SubjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{bound}, nil
}

func decideUnbind(state campaign.State, act caller.Caller, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[participant.Unbind](envelope)
	if err != nil {
		return nil, err
	}
	record, err := requireActiveParticipant(state, message.ParticipantID)
	if err != nil {
		return nil, err
	}
	if err := authorizeSeatManagement(state, act, record, true, authz.CapabilityUnbindParticipant); err != nil {
		return nil, err
	}
	if record.Access == participant.AccessOwner {
		return nil, errs.FailedPreconditionf("campaign owner participant cannot be unbound")
	}
	if record.SubjectID == "" {
		return nil, errs.FailedPreconditionf("participant %s is not bound", record.ID)
	}
	unbound, err := event.NewEnvelope(
		participant.UnboundEventSpec,
		envelope.CampaignID,
		participant.Unbound{ParticipantID: record.ID},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{unbound}, nil
}
