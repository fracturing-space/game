package service

import (
	"context"
	"testing"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/participant"
)

func TestExecuteCreateStoresOneLiveCommit(t *testing.T) {
	svc := newTestService(t)

	result, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create) error: %v", err)
	}
	if got, want := len(result.Events), 2; got != want {
		t.Fatalf("planned events len = %d, want %d", got, want)
	}
	if got, want := len(result.StoredEvents), 2; got != want {
		t.Fatalf("stored events len = %d, want %d", got, want)
	}
	if got, want := result.StoredEvents[0].Seq, uint64(1); got != want {
		t.Fatalf("seq = %d, want %d", got, want)
	}
	if got, want := result.StoredEvents[0].CommitSeq, uint64(1); got != want {
		t.Fatalf("commit seq = %d, want %d", got, want)
	}
	if got, want := result.StoredEvents[0].RecordedAt, fixedRecordTime; !got.Equal(want) {
		t.Fatalf("recorded at = %v, want %v", got, want)
	}
	if got, want := result.State.PlayState, campaign.PlayStateSetup; got != want {
		t.Fatalf("state play state = %q, want %q", got, want)
	}
	if got, want := len(result.State.Participants), 1; got != want {
		t.Fatalf("participants len = %d, want %d", got, want)
	}
	if got, want := result.State.Participants[0].Access, participant.AccessOwner; got != want {
		t.Fatalf("owner access = %q, want %q", got, want)
	}
	if got := result.State.AIAgentID; got != "" {
		t.Fatalf("ai agent id = %q, want empty", got)
	}
}

func TestSetAndClearCampaignAIBinding(t *testing.T) {
	fixture := newCreatedCampaignFixture(t)

	bindResult, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message:    campaign.AIBind{AIAgentID: "agent-7"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(ai bind) error: %v", err)
	}
	if got, want := bindResult.State.AIAgentID, "agent-7"; got != want {
		t.Fatalf("state ai agent id = %q, want %q", got, want)
	}

	inspection, err := fixture.Service.Inspect(context.Background(), fixture.OwnerCaller, fixture.CampaignID)
	if err != nil {
		t.Fatalf("Inspect(after bind) error: %v", err)
	}
	if got, want := inspection.State.AIAgentID, "agent-7"; got != want {
		t.Fatalf("inspection ai agent id = %q, want %q", got, want)
	}

	clearResult, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message:    campaign.AIUnbind{},
	})
	if err != nil {
		t.Fatalf("CommitCommand(ai unbind) error: %v", err)
	}
	if got := clearResult.State.AIAgentID; got != "" {
		t.Fatalf("state ai agent id after clear = %q, want empty", got)
	}
}

func TestCreateCharacterStoresOneLiveCommit(t *testing.T) {
	fixture := newCreatedCampaignFixture(t)

	result, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message: character.Create{
			ParticipantID: fixture.ParticipantID,
			Name:          " luna "},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create character) error: %v", err)
	}
	if got, want := len(result.Events), 1; got != want {
		t.Fatalf("planned events len = %d, want %d", got, want)
	}
	if got, want := len(result.StoredEvents), 1; got != want {
		t.Fatalf("stored events len = %d, want %d", got, want)
	}
	if got, want := len(result.State.Characters), 1; got != want {
		t.Fatalf("characters len = %d, want %d", got, want)
	}
	if got, want := result.State.Characters[0].ParticipantID, fixture.ParticipantID; got != want {
		t.Fatalf("participant id = %q, want %q", got, want)
	}
	if got, want := result.State.Characters[0].Name, "luna"; got != want {
		t.Fatalf("character name = %q, want %q", got, want)
	}

	inspection, err := fixture.Service.Inspect(context.Background(), fixture.OwnerCaller, fixture.CampaignID)
	if err != nil {
		t.Fatalf("Inspect(after create character) error: %v", err)
	}
	if got, want := len(inspection.State.Characters), 1; got != want {
		t.Fatalf("inspection characters len = %d, want %d", got, want)
	}
	if got, want := inspection.State.Characters[0].ID, "char-1"; got != want {
		t.Fatalf("character id = %q, want %q", got, want)
	}
}

func TestExecuteRejectsInvalidCreateWithoutMutation(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "   ", OwnerName: "louis"},
	}); err == nil {
		t.Fatal("CommitCommand(create) should reject blank campaign names")
	}

	if _, err := svc.Inspect(context.Background(), defaultCaller(), "camp-1"); err == nil {
		t.Fatal("Inspect(camp-1) should fail because rejected commands must not mutate the journal")
	}
}

func TestJoinRejectedInPlay(t *testing.T) {
	fixture := newInPlayCampaignFixture(t)

	if _, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message: participant.Join{
			Name: "zoe", Access: participant.AccessMember},
	}); err == nil {
		t.Fatal("CommitCommand(create participant in play) error = nil, want failure")
	}
}
