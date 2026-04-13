package authz

import (
	"errors"
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/participant"
)

func TestDeniedErrorAndIsDenied(t *testing.T) {
	t.Parallel()

	err := &DeniedError{Capability: CapabilityReadCampaign, Reason: "boom"}
	if got, want := err.Error(), "read_campaign denied: boom"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !IsDenied(err) {
		t.Fatal("IsDenied() = false, want true")
	}
	if IsDenied(errors.New("boom")) {
		t.Fatal("IsDenied() = true, want false")
	}
	if got, want := (&DeniedError{Capability: CapabilityAuthenticated}).Error(), "authenticated denied"; got != want {
		t.Fatalf("Error(blank reason) = %q, want %q", got, want)
	}
	if got := (*DeniedError)(nil).Error(); got != "" {
		t.Fatalf("Error(nil) = %q, want empty string", got)
	}
}

func TestRequireAuthenticatedCommands(t *testing.T) {
	t.Parallel()

	if err := RequireCreateCampaign(caller.MustNewSubject("subject-1")); err != nil {
		t.Fatalf("RequireCreateCampaign() error = %v", err)
	}
	if err := RequireBindParticipant(caller.MustNewSubject("subject-1")); err != nil {
		t.Fatalf("RequireBindParticipant() error = %v", err)
	}
	if err := RequireCreateCharacter(caller.MustNewSubject("subject-1")); err != nil {
		t.Fatalf("RequireCreateCharacter() error = %v", err)
	}
	if err := RequireCreateCampaign(caller.Caller{}); !IsDenied(err) {
		t.Fatalf("RequireCreateCampaign() error = %v, want denied", err)
	}
}

func TestRequireAuthenticatedWrappers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(caller.Caller) error
	}{
		{name: "caller", call: RequireCaller},
		{name: "unbind participant", call: RequireUnbindParticipant},
		{name: "update character", call: RequireUpdateCharacter},
		{name: "delete character", call: RequireDeleteCharacter},
	}
	for _, test := range tests {
		if err := test.call(caller.MustNewSubject("subject-1")); err != nil {
			t.Fatalf("%s(valid) error = %v", test.name, err)
		}
		if err := test.call(caller.Caller{}); !IsDenied(err) {
			t.Fatalf("%s(invalid) error = %v, want denied", test.name, err)
		}
	}
}

func TestRequireOwnerAccess(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.Participants["owner-1"] = participant.Record{
		ID:        "owner-1",
		Name:      "Owner",
		Access:    participant.AccessOwner,
		SubjectID: "subject-owner",
		Active:    true,
	}
	state.Participants["member-1"] = participant.Record{
		ID:        "member-1",
		Name:      "Member",
		Access:    participant.AccessMember,
		SubjectID: "subject-member",
		Active:    true,
	}

	if err := RequirePausePlay(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("RequirePausePlay(owner) error = %v", err)
	}
	if err := RequireManageAIBinding(caller.MustNewSubject("subject-member"), state); !IsDenied(err) {
		t.Fatalf("RequireManageAIBinding(member) error = %v, want denied", err)
	}
	if err := RequireStartSession(caller.MustNewSubject("subject-missing"), state); !IsDenied(err) {
		t.Fatalf("RequireStartSession(unbound) error = %v, want denied", err)
	}
	if err := RequireEndSession(caller.MustNewSubject("subject-owner"), state); err != nil {
		t.Fatalf("RequireEndSession(owner) error = %v", err)
	}
	if err := RequireResumePlay(caller.MustNewSubject("subject-member"), state); !IsDenied(err) {
		t.Fatalf("RequireResumePlay(member) error = %v, want denied", err)
	}
	if err := RequireEndPlay(caller.Caller{}, state); !IsDenied(err) {
		t.Fatalf("RequireEndPlay(missing caller) error = %v, want denied", err)
	}
}

func TestRequireReadCampaign(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.Participants["part-1"] = participant.Record{
		ID:        "part-1",
		Name:      "Member",
		Access:    participant.AccessMember,
		SubjectID: "subject-1",
		Active:    true,
	}

	if err := RequireReadCampaign(caller.MustNewSubject("subject-1"), state); err != nil {
		t.Fatalf("RequireReadCampaign(bound) error = %v", err)
	}
	if err := RequireReadCampaign(caller.MustNewSubject("subject-2"), state); !IsDenied(err) {
		t.Fatalf("RequireReadCampaign(unbound) error = %v, want denied", err)
	}
	if err := RequireReadCampaign(caller.MustNewAIAgent("agent-1"), state); !IsDenied(err) {
		t.Fatalf("RequireReadCampaign(ai caller) error = %v, want denied", err)
	}
}

func TestRequireAIGMAccess(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.AIAgentID = "agent-1"

	if err := RequireCreateScene(caller.MustNewAIAgent("agent-1"), state); err != nil {
		t.Fatalf("RequireCreateScene(ai gm) error = %v", err)
	}
	if err := RequirePlanCommands(caller.MustNewAIAgent("agent-1"), state); err != nil {
		t.Fatalf("RequirePlanCommands(ai gm) error = %v", err)
	}
	if err := RequireCreateScene(caller.MustNewSubject("subject-1"), state); !IsDenied(err) {
		t.Fatalf("RequireCreateScene(subject) error = %v, want denied", err)
	}
	if err := RequireCreateScene(caller.MustNewAIAgent("agent-2"), state); !IsDenied(err) {
		t.Fatalf("RequireCreateScene(mismatched ai) error = %v, want denied", err)
	}

	state.AIAgentID = ""
	if err := RequirePlanCommands(caller.MustNewAIAgent("agent-1"), state); !IsDenied(err) {
		t.Fatalf("RequirePlanCommands(unbound campaign ai) error = %v, want denied", err)
	}
}
