package readiness

import (
	"strings"
	"testing"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/participant"
)

func TestEvaluatePlay(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	report := EvaluatePlay(state)
	if got, want := report.Blockers[0].Code, RejectionCodePlayReadinessAIAgentRequired; got != want {
		t.Fatalf("first blocker = %q, want %q", got, want)
	}

	state.AIAgentID = "agent-1"
	state.Participants["owner-1"] = participant.Record{
		ID:        "owner-1",
		Name:      "Owner",
		Access:    participant.AccessOwner,
		SubjectID: "subject-owner",
		Active:    true,
	}
	report = EvaluatePlay(state)
	if got, want := report.Blockers[0].Code, RejectionCodePlayReadinessPlayerCharacterRequired; got != want {
		t.Fatalf("player blocker = %q, want %q", got, want)
	}

	state.Characters["char-1"] = character.Record{
		ID:            "char-1",
		ParticipantID: "owner-1",
		Name:          "Luna",
		Active:        true,
	}
	if report := EvaluatePlay(state); !report.Ready() {
		t.Fatalf("EvaluatePlay(ready) blockers = %v, want none", report.Blockers)
	}
}

func TestValidateReportAndNormalizeIDs(t *testing.T) {
	t.Parallel()

	if err := ValidateReport(Report{Blockers: []Blocker{{
		Code:    "X",
		Message: "configure ai",
		Action: Action{
			ResponsibleParticipantIDs: []string{"owner-1"},
			ResolutionKind:            ResolutionKindConfigureAIAgent,
		},
	}}}); err != nil {
		t.Fatalf("ValidateReport(valid) error = %v", err)
	}

	if got := normalizeIDs([]string{"owner-2", "", "owner-1", "owner-2"}); strings.Join(got, ",") != "owner-1,owner-2" {
		t.Fatalf("normalizeIDs() = %v, want sorted unique ids", got)
	}
}

func TestActiveCharacterCountsByParticipant(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.Characters["char-1"] = character.Record{ID: "char-1", ParticipantID: "part-1", Active: true}
	state.Characters["char-2"] = character.Record{ID: "char-2", ParticipantID: "part-1", Active: true}
	state.Characters["char-3"] = character.Record{ID: "char-3", ParticipantID: "part-2"}

	counts := activeCharacterCountsByParticipant(state)
	if got, want := counts["part-1"], 2; got != want {
		t.Fatalf("count = %d, want %d", got, want)
	}
	if _, ok := counts["part-2"]; ok {
		t.Fatalf("count for inactive participant owner = %d, want omitted", counts["part-2"])
	}
	if got := activeCharacterCountsByParticipant(campaign.NewState()); got != nil {
		t.Fatalf("activeCharacterCountsByParticipant(empty) = %v, want nil", got)
	}
}

func TestReadinessHelpersAndErrors(t *testing.T) {
	t.Parallel()

	if got, want := (&Rejection{Message: " blocked "}).Error(), "blocked"; got != want {
		t.Fatalf("Rejection.Error() = %q, want %q", got, want)
	}
	if got := (*Rejection)(nil).Error(); got != "" {
		t.Fatalf("Rejection(nil).Error() = %q, want empty", got)
	}
	if _, ok := AsRejection(nil); ok {
		t.Fatal("AsRejection(nil) = true, want false")
	}

	index := indexedParticipants{ownerIDs: []string{"owner-2", "owner-1"}}
	if action := ownerManageParticipantsAction(index); action.ResolutionKind != ResolutionKindManageParticipants {
		t.Fatalf("ownerManageParticipantsAction() = %+v, want manage participants", action)
	}

	if got, want := (playerRecord{Name: "Owner"}).displayName(), "Owner"; got != want {
		t.Fatalf("displayName(name) = %q, want %q", got, want)
	}
	if got, want := (playerRecord{ID: "part-1"}).displayName(), "part-1"; got != want {
		t.Fatalf("displayName(id fallback) = %q, want %q", got, want)
	}

	action := cloneAction(Action{
		ResponsibleParticipantIDs: []string{"owner-2", "", "owner-1", "owner-2"},
		ResolutionKind:            ResolutionKindInvitePlayer,
	})
	if got, want := strings.Join(action.ResponsibleParticipantIDs, ","), "owner-1,owner-2"; got != want {
		t.Fatalf("cloneAction() ids = %q, want %q", got, want)
	}
}
