package service

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/engine"
	modulecampaign "github.com/fracturing-space/game/internal/modules/campaign"
	"github.com/fracturing-space/game/internal/participant"
)

func TestDifferentRecordClocksKeepSameDomainResult(t *testing.T) {
	svcA := newTestServiceAt(t, fixedRecordTime)
	svcB := newTestServiceAt(t, fixedRecordTime.Add(24*time.Hour))
	createCmd := command.Envelope{Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"}}

	createA, err := svcA.CommitCommand(context.Background(), defaultCaller(), createCmd)
	if err != nil {
		t.Fatalf("CommitCommand(create A) error: %v", err)
	}
	createB, err := svcB.CommitCommand(context.Background(), defaultCaller(), createCmd)
	if err != nil {
		t.Fatalf("CommitCommand(create B) error: %v", err)
	}

	joinCmdA := command.Envelope{CampaignID: createA.State.ID, Message: participant.Join{
		Name: "louis", Access: participant.AccessMember}}
	joinCmdB := command.Envelope{CampaignID: createB.State.ID, Message: participant.Join{
		Name: "louis", Access: participant.AccessMember}}
	joinA, err := svcA.CommitCommand(context.Background(), defaultCaller(), joinCmdA)
	if err != nil {
		t.Fatalf("CommitCommand(join A) error: %v", err)
	}
	joinB, err := svcB.CommitCommand(context.Background(), defaultCaller(), joinCmdB)
	if err != nil {
		t.Fatalf("CommitCommand(join B) error: %v", err)
	}
	if !reflect.DeepEqual(joinA.State, joinB.State) {
		t.Fatalf("state differs:\nA=%#v\nB=%#v", joinA.State, joinB.State)
	}
	for index := range joinA.Events {
		if !reflect.DeepEqual(joinA.Events[index], joinB.Events[index]) {
			t.Fatalf("planned event %d differs:\nA=%#v\nB=%#v", index, joinA.Events[index], joinB.Events[index])
		}
	}
	if joinA.StoredEvents[0].RecordedAt.Equal(joinB.StoredEvents[0].RecordedAt) {
		t.Fatal("recorded_at should differ across clocks")
	}
}

func TestNewRejectsDuplicateCommandRegistration(t *testing.T) {
	_, err := BuildManifest([]engine.Module{
		modulecampaign.New(),
		duplicateModule{},
	})
	if err == nil {
		t.Fatal("BuildManifest() should fail when modules register duplicate command types")
	}
	if !strings.Contains(err.Error(), "command type already routed") && !strings.Contains(err.Error(), "command type already registered") {
		t.Fatalf("error = %v, want duplicate registration failure", err)
	}
}
