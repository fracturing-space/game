package participant

import (
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
)

func decideJoin(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[participant.Join](envelope)
	if err != nil {
		return nil, err
	}
	if message.SubjectID == "" && message.Access == participant.AccessOwner {
		message.SubjectID = act.SubjectID
	}
	if message.SubjectID != "" && campaign.HasBoundSubject(state, message.SubjectID) {
		return nil, errs.Conflictf("subject is already bound in campaign")
	}
	participantID, err := ids("part")
	if err != nil {
		return nil, err
	}
	joined, err := event.NewEnvelope(
		participant.JoinedEventSpec,
		envelope.CampaignID,
		participant.Joined{
			ParticipantID: participantID,
			Name:          message.Name,
			Access:        message.Access,
			SubjectID:     message.SubjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{joined}, nil
}

func decideUpdate(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[participant.Update](envelope)
	if err != nil {
		return nil, err
	}
	record, err := requireActiveParticipant(state, message.ParticipantID)
	if err != nil {
		return nil, err
	}
	if record.Access == participant.AccessOwner {
		if message.Access != record.Access {
			return nil, errs.FailedPreconditionf("owner participant identity cannot be reassigned")
		}
	}
	if message.Access == participant.AccessOwner && record.Access != participant.AccessOwner {
		return nil, errs.FailedPreconditionf("campaign owner participant cannot be reassigned")
	}
	if err := participant.ValidateJoin(participant.Join{
		Name:      message.Name,
		Access:    message.Access,
		SubjectID: record.SubjectID,
	}); err != nil {
		return nil, err
	}
	updated, err := event.NewEnvelope(
		participant.UpdatedEventSpec,
		envelope.CampaignID,
		participant.Updated(message),
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{updated}, nil
}

func decideLeave(state campaign.State, envelope command.Envelope) ([]event.Envelope, error) {
	if !state.Exists {
		return nil, errs.NotFoundf("campaign does not exist")
	}
	message, err := command.MessageAs[participant.Leave](envelope)
	if err != nil {
		return nil, err
	}
	record, err := requireActiveParticipant(state, message.ParticipantID)
	if err != nil {
		return nil, err
	}
	if record.Access == participant.AccessOwner {
		return nil, errs.FailedPreconditionf("campaign owner participant cannot be removed")
	}
	for _, next := range state.Characters {
		if !next.Active {
			continue
		}
		if next.ParticipantID == record.ID {
			return nil, errs.FailedPreconditionf("participant %s still owns active characters", record.ID)
		}
	}
	if activeSession := state.ActiveSession(); activeSession != nil {
		// Participant mutation is setup-only, so an active session here indicates a
		// violated session-state invariant rather than an expected play-time path.
		for _, assignment := range activeSession.CharacterControllers {
			if assignment.ParticipantID == record.ID {
				return nil, errs.FailedPreconditionf("participant %s still has active session controller responsibilities", record.ID)
			}
		}
	}
	left, err := event.NewEnvelope(
		participant.LeftEventSpec,
		envelope.CampaignID,
		participant.Left{ParticipantID: record.ID},
	)
	if err != nil {
		return nil, err
	}
	return []event.Envelope{left}, nil
}
