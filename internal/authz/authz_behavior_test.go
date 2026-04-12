package authz

import (
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/participant"
)

func TestRequireOwnerAccessCommands(t *testing.T) {
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

	tests := []struct {
		name string
		fn   func(caller.Caller, campaign.State) error
	}{
		{name: "update campaign", fn: RequireUpdateCampaign},
		{name: "create participant", fn: RequireCreateParticipant},
		{name: "update participant", fn: RequireUpdateParticipant},
		{name: "delete participant", fn: RequireDeleteParticipant},
		{name: "begin play", fn: RequireBeginPlay},
		{name: "end play", fn: RequireEndPlay},
	}

	for _, tt := range tests {
		if err := tt.fn(caller.MustNewSubject("subject-owner"), state); err != nil {
			t.Fatalf("%s owner error = %v", tt.name, err)
		}
		if err := tt.fn(caller.MustNewSubject("subject-member"), state); !IsDenied(err) {
			t.Fatalf("%s member error = %v, want denied", tt.name, err)
		}
	}
}

func TestRequireAIGMCommands(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.AIAgentID = "agent-1"

	tests := []struct {
		name string
		fn   func(caller.Caller, campaign.State) error
	}{
		{name: "create scene", fn: RequireCreateScene},
		{name: "activate scene", fn: RequireActivateScene},
		{name: "end scene", fn: RequireEndScene},
		{name: "replace scene cast", fn: RequireReplaceSceneCast},
	}

	for _, tt := range tests {
		if err := tt.fn(caller.MustNewAIAgent("agent-1"), state); err != nil {
			t.Fatalf("%s bound ai error = %v", tt.name, err)
		}
		if err := tt.fn(caller.MustNewSubject("subject-owner"), state); !IsDenied(err) {
			t.Fatalf("%s subject error = %v, want denied", tt.name, err)
		}
		if err := tt.fn(caller.MustNewAIAgent("agent-2"), state); !IsDenied(err) {
			t.Fatalf("%s mismatched ai error = %v, want denied", tt.name, err)
		}
	}
}
