package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/scene"
)

func TestPlanCommandsAndExecutePlan(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)

	plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Climax",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	}})
	if err != nil {
		t.Fatalf("PlanCommands() error = %v", err)
	}
	if plan.Token == "" {
		t.Fatal("plan token = empty, want token")
	}
	if got, want := plan.CampaignID, fixture.CampaignID; got != want {
		t.Fatalf("plan campaign id = %q, want %q", got, want)
	}
	if got, want := len(plan.Commits), 1; got != want {
		t.Fatalf("plan commits len = %d, want %d", got, want)
	}
	if got, want := len(plan.State.Scenes), 2; got != want {
		t.Fatalf("planned scenes len = %d, want %d", got, want)
	}

	liveBefore, err := fixture.Service.Inspect(context.Background(), fixture.OwnerCaller, fixture.CampaignID)
	if err != nil {
		t.Fatalf("Inspect(before execute plan) error = %v", err)
	}
	if got, want := len(liveBefore.State.Scenes), 1; got != want {
		t.Fatalf("live scenes before execute plan = %d, want %d", got, want)
	}

	executed, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token)
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if got, want := executed.CampaignID, fixture.CampaignID; got != want {
		t.Fatalf("executed campaign id = %q, want %q", got, want)
	}
	if got, want := len(executed.State.Scenes), 2; got != want {
		t.Fatalf("executed scenes len = %d, want %d", got, want)
	}

	liveAfter, err := fixture.Service.Inspect(context.Background(), fixture.OwnerCaller, fixture.CampaignID)
	if err != nil {
		t.Fatalf("Inspect(after execute plan) error = %v", err)
	}
	if got, want := len(liveAfter.State.Scenes), 2; got != want {
		t.Fatalf("live scenes after execute plan = %d, want %d", got, want)
	}
}

func TestExecutePlanRejectsChangedCampaignHead(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)
	plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Climax",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	}})
	if err != nil {
		t.Fatalf("PlanCommands() error = %v", err)
	}

	if _, err := fixture.Service.CommitCommand(context.Background(), fixture.GMCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Interlude",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	}); err != nil {
		t.Fatalf("CommitCommand(live scene create) error = %v", err)
	}

	if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); !errs.Is(err, errs.KindConflict) {
		t.Fatalf("ExecutePlan(stale) error = %v, want conflict", err)
	}
}

func TestPlanCommandsRejectsNonGMAndInactiveCampaigns(t *testing.T) {
	t.Parallel()

	t.Run("owner cannot plan gm commands", func(t *testing.T) {
		fixture := newActiveSessionCampaignFixture(t)
		_, err := fixture.Service.PlanCommands(context.Background(), fixture.OwnerCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "Climax",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}})
		if !authz.IsDenied(err) {
			t.Fatalf("PlanCommands(owner) error = %v, want denied", err)
		}
	})

	t.Run("gm cannot plan while not active", func(t *testing.T) {
		fixture := newPlayReadyCampaignFixture(t)
		_, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "Climax",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}})
		if !errs.Is(err, errs.KindFailedPrecondition) {
			t.Fatalf("PlanCommands(inactive) error = %v, want failed precondition", err)
		}
	})
}

func TestPlanCommandsRejectsUnsupportedCommands(t *testing.T) {
	t.Parallel()

	fixture := newGMEnabledStubCampaignFixture(t, stubServiceModule{})

	_, err := fixture.Service.PlanCommands(context.Background(), caller.MustNewAIAgent("agent-1"), []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message:    testLiveCommand{Name: "next"},
	}})
	if !errs.Is(err, errs.KindInvalidArgument) {
		t.Fatalf("PlanCommands(unsupported command) error = %v, want invalid argument", err)
	}
}

func TestPlanCommandsRejectsInvalidBatches(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)
	second, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		Message: campaign.Create{Name: "Winter Dawn", OwnerName: "louis"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(second create) error = %v", err)
	}
	secondCampaignID := second.State.ID

	cases := []struct {
		name     string
		commands []command.Envelope
	}{
		{name: "empty"},
		{
			name: "new campaign command",
			commands: []command.Envelope{{
				Message: campaign.Create{Name: "Shadow", OwnerName: "louis"},
			}},
		},
		{
			name: "mixed campaigns",
			commands: []command.Envelope{
				{CampaignID: fixture.CampaignID, Message: scene.End{SceneID: fixture.SceneID}},
				{CampaignID: secondCampaignID, Message: scene.End{SceneID: fixture.SceneID}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, tc.commands); !errs.Is(err, errs.KindInvalidArgument) {
				t.Fatalf("PlanCommands(%s) error = %v, want invalid argument", tc.name, err)
			}
		})
	}
}

func TestExecutePlanWrongActorDoesNotConsumeToken(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)
	plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Climax",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	}})
	if err != nil {
		t.Fatalf("PlanCommands() error = %v", err)
	}

	if _, err := fixture.Service.ExecutePlan(context.Background(), caller.MustNewAIAgent("agent-2"), plan.Token); !errs.Is(err, errs.KindFailedPrecondition) {
		t.Fatalf("ExecutePlan(wrong caller) error = %v, want failed precondition", err)
	}
	if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); err != nil {
		t.Fatalf("ExecutePlan(correct caller) error = %v", err)
	}
}

func TestPlanCommandsRejectsSecondActivePlanAndSweepsExpiredPlans(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)
	clock := &mutableClock{at: fixedRecordTime}
	fixture.Service.recordClock = clock
	fixture.Service.planTTL = time.Minute

	first, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Climax",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	}})
	if err != nil {
		t.Fatalf("PlanCommands(first) error = %v", err)
	}
	if got, want := len(fixture.Service.planStore().items), 1; got != want {
		t.Fatalf("plan store size after first plan = %d, want %d", got, want)
	}

	if _, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message:    scene.End{SceneID: fixture.SceneID},
	}}); !errs.Is(err, errs.KindConflict) {
		t.Fatalf("PlanCommands(second active plan) error = %v, want conflict", err)
	}

	clock.at = clock.at.Add(2 * time.Minute)
	second, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message:    scene.End{SceneID: fixture.SceneID},
	}})
	if err != nil {
		t.Fatalf("PlanCommands(after expiry) error = %v", err)
	}
	if second.Token == first.Token {
		t.Fatalf("second plan token = %q, want fresh token", second.Token)
	}
}

func TestExecutePlanRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)
	clock := &mutableClock{at: fixedRecordTime}
	fixture.Service.recordClock = clock
	fixture.Service.planTTL = time.Minute

	plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Climax",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	}})
	if err != nil {
		t.Fatalf("PlanCommands() error = %v", err)
	}

	clock.at = clock.at.Add(2 * time.Minute)
	if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); !errs.Is(err, errs.KindNotFound) {
		t.Fatalf("ExecutePlan(expired) error = %v, want not found", err)
	}
}

func TestExecutePlanConsumesTokenOnSuccess(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)
	plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Climax",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	}})
	if err != nil {
		t.Fatalf("PlanCommands() error = %v", err)
	}

	if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); !errs.Is(err, errs.KindNotFound) {
		t.Fatalf("ExecutePlan(reused token) error = %v, want not found", err)
	}
}

func TestExecutePlanPropagatesLoadAppendAndPersistFailures(t *testing.T) {
	t.Parallel()

	t.Run("load failure", func(t *testing.T) {
		fixture := newActiveSessionCampaignFixture(t)
		plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "Climax",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}})
		if err != nil {
			t.Fatalf("PlanCommands() error = %v", err)
		}

		fixture.Service.store = failingJournal{
			base:    fixture.Service.store,
			listErr: errors.New("journal list failed"),
		}
		if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); err == nil || !strings.Contains(err.Error(), "journal list failed") {
			t.Fatalf("ExecutePlan(load failure) error = %v, want journal list failure", err)
		}
	})

	t.Run("append failure", func(t *testing.T) {
		fixture := newActiveSessionCampaignFixture(t)
		plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "Climax",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}})
		if err != nil {
			t.Fatalf("PlanCommands() error = %v", err)
		}

		fixture.Service.store = failingJournal{
			base:      fixture.Service.store,
			appendErr: errors.New("append failed"),
		}
		if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); err == nil || !strings.Contains(err.Error(), "append failed") {
			t.Fatalf("ExecutePlan(append failure) error = %v, want append failure", err)
		}
	})

	t.Run("persist failure", func(t *testing.T) {
		fixture := newActiveSessionCampaignFixture(t)
		plan, err := fixture.Service.PlanCommands(context.Background(), fixture.GMCaller, []command.Envelope{{
			CampaignID: fixture.CampaignID,
			Message: scene.Create{
				Name:         "Climax",
				CharacterIDs: []string{fixture.OwnerCharacterID},
			},
		}})
		if err != nil {
			t.Fatalf("PlanCommands() error = %v", err)
		}

		fixture.Service.projections = failingProjectionStore{
			base:              fixture.Service.projections,
			saveProjectionErr: errors.New("projection save failed"),
		}
		if _, err := fixture.Service.ExecutePlan(context.Background(), fixture.GMCaller, plan.Token); err == nil || !strings.Contains(err.Error(), "projection save failed") {
			t.Fatalf("ExecutePlan(persist failure) error = %v, want projection save failure", err)
		}
	})
}

type mutableClock struct {
	at time.Time
}

func (c *mutableClock) Now() time.Time {
	return c.at
}

func callerWithSubject(subjectID string) caller.Caller {
	return caller.MustNewSubject(subjectID)
}
