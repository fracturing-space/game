package readiness

import (
	"testing"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/participant"
)

func TestHelpersAndValidation(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.AIAgentID = "agent-1"
	state.Participants["owner-1"] = participant.Record{
		ID:        "owner-1",
		Access:    participant.AccessOwner,
		SubjectID: "subject-owner",
		Active:    true,
	}
	state.Characters["char-1"] = character.Record{
		ID:            "char-1",
		ParticipantID: "owner-1",
		Active:        true,
	}

	index := participantIndex(state)
	if got, want := len(index.ownerIDs), 1; got != want {
		t.Fatalf("ownerIDs len = %d, want %d", got, want)
	}
	if got, want := len(index.boundParticipants), 1; got != want {
		t.Fatalf("boundParticipants len = %d, want %d", got, want)
	}

	if err := validateAction(Action{
		ResponsibleParticipantIDs: []string{"owner-1"},
		ResolutionKind:            ResolutionKindCreateCharacter,
		TargetParticipantID:       "part-1",
	}); err != nil {
		t.Fatalf("validateAction(valid) error = %v", err)
	}

	if _, ok := AsRejection(EvaluatePlayTransition(campaign.NewState())); !ok {
		t.Fatal("AsRejection() = false, want true")
	}
}

func TestValidateReportAndActionBranches(t *testing.T) {
	t.Parallel()

	if err := ValidateReport(Report{Blockers: []Blocker{{Message: "missing code"}}}); err == nil {
		t.Fatal("ValidateReport(missing code) error = nil, want failure")
	}
	if err := ValidateReport(Report{Blockers: []Blocker{{Code: "X"}}}); err == nil {
		t.Fatal("ValidateReport(missing message) error = nil, want failure")
	}
	if err := ValidateReport(Report{Blockers: []Blocker{{
		Code:    "X",
		Message: "invalid action",
		Action: Action{
			ResponsibleParticipantIDs: []string{" owner-1 "},
			ResolutionKind:            ResolutionKindConfigureAIAgent,
		},
	}}}); err == nil {
		t.Fatal("ValidateReport(invalid action) error = nil, want failure")
	}

	if err := validateAction(Action{ResolutionKind: ResolutionKind("broken")}); err == nil {
		t.Fatal("validateAction(invalid kind) error = nil, want failure")
	}
	if err := validateAction(Action{
		ResponsibleParticipantIDs: []string{"owner-2", "owner-1"},
		ResolutionKind:            ResolutionKindManageParticipants,
		TargetParticipantID:       "part-1",
	}); err == nil {
		t.Fatal("validateAction(target not allowed) error = nil, want failure")
	}
	if err := validateAction(Action{
		ResponsibleParticipantIDs: []string{"owner-2", "owner-1"},
		ResolutionKind:            ResolutionKindCreateCharacter,
	}); err == nil {
		t.Fatal("validateAction(missing target) error = nil, want failure")
	}
	if err := validateAction(Action{
		ResponsibleParticipantIDs: []string{"owner-1", "owner-2"},
		ResolutionKind:            ResolutionKindInvitePlayer,
	}); err != nil {
		t.Fatalf("validateAction(valid invite) error = %v", err)
	}
}
