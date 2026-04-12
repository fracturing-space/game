package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

var (
	fixedRecordTime      = time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)
	serviceTestClockTime = fixedRecordTime
)

type fixedClock struct {
	at time.Time
}

func (c fixedClock) Now() time.Time {
	return c.at
}

type realCampaignFixture struct {
	Service          *Service
	CampaignID       string
	OwnerCaller      caller.Caller
	GMCaller         caller.Caller
	ParticipantID    string
	OwnerCharacterID string
	SessionID        string
	SceneID          string
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceAt(t, fixedRecordTime)
}

func newTestServiceWithLogger(t *testing.T, logger *slog.Logger) *Service {
	t.Helper()
	return newTestServiceAtWithLogger(t, fixedRecordTime, logger)
}

func newTestServiceAt(t *testing.T, at time.Time) *Service {
	t.Helper()
	return newTestServiceAtWithLogger(t, at, nil)
}

func newTestServiceAtWithLogger(t *testing.T, at time.Time, logger *slog.Logger) *Service {
	t.Helper()

	manifest, err := BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error: %v", err)
	}
	svc, err := New(Config{
		Manifest:        manifest,
		IDs:             newSequentialIDAllocator(),
		RecordClock:     fixedClock{at: at},
		Journal:         newTestMemoryStore(),
		ProjectionStore: newTestProjectionStore(),
		ArtifactStore:   newTestArtifactStore(),
		Logger:          logger,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return svc
}

func newCreatedCampaignFixture(t *testing.T) realCampaignFixture {
	t.Helper()

	fixture := realCampaignFixture{
		Service:     newTestService(t),
		OwnerCaller: defaultCaller(),
	}
	result, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create) error: %v", err)
	}
	fixture.CampaignID = result.State.ID
	fixture.ParticipantID = mustParticipantID(t, result.State)
	return fixture
}

func newPlayReadyCampaignFixture(t *testing.T) realCampaignFixture {
	t.Helper()

	fixture := newCreatedCampaignFixture(t)
	if _, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message:    campaign.AIBind{AIAgentID: "agent-7"},
	}); err != nil {
		t.Fatalf("CommitCommand(ai bind) error: %v", err)
	}
	createCharacterResult, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message: character.Create{
			ParticipantID: fixture.ParticipantID,
			Name:          "luna"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create character) error: %v", err)
	}
	fixture.OwnerCharacterID = mustCharacterID(t, createCharacterResult.State, "luna")

	startResult, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message:    session.Start{},
	})
	if err != nil {
		t.Fatalf("CommitCommand(start session) error: %v", err)
	}
	if startResult.State.ActiveSession() == nil {
		t.Fatal("CommitCommand(start session) active session = nil, want session")
	}
	fixture.SessionID = startResult.State.ActiveSession().ID
	fixture.GMCaller = caller.MustNewAIAgent("agent-7")
	return fixture
}

func newActiveSessionCampaignFixture(t *testing.T) realCampaignFixture {
	t.Helper()

	fixture := newPlayReadyCampaignFixture(t)
	if _, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message:    campaign.PlayBegin{},
	}); err != nil {
		t.Fatalf("CommitCommand(begin play) error: %v", err)
	}

	createSceneResult, err := fixture.Service.CommitCommand(context.Background(), fixture.GMCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message: scene.Create{
			Name:         "Opening Scene",
			CharacterIDs: []string{fixture.OwnerCharacterID},
		},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create scene) error: %v", err)
	}
	fixture.SceneID = mustSceneID(t, createSceneResult.State, "Opening Scene")

	if _, err := fixture.Service.CommitCommand(context.Background(), fixture.GMCaller, command.Envelope{
		CampaignID: fixture.CampaignID,
		Message:    scene.Activate{SceneID: fixture.SceneID},
	}); err != nil {
		t.Fatalf("CommitCommand(activate scene) error: %v", err)
	}
	return fixture
}

func newInPlayCampaignFixture(t *testing.T) realCampaignFixture {
	t.Helper()

	return newActiveSessionCampaignFixture(t)
}

func mustParticipantID(t *testing.T, snapshot campaign.Snapshot) string {
	t.Helper()

	for _, record := range snapshot.Participants {
		if record.Access == participant.AccessOwner {
			return record.ID
		}
	}
	t.Fatal("owner participant not found in snapshot")
	return ""
}

func mustCharacterID(t *testing.T, snapshot campaign.Snapshot, name string) string {
	t.Helper()

	for _, record := range snapshot.Characters {
		if record.Name == name {
			return record.ID
		}
	}
	t.Fatalf("character %q not found in snapshot", name)
	return ""
}

func mustSceneID(t *testing.T, snapshot campaign.Snapshot, name string) string {
	t.Helper()

	for _, record := range snapshot.Scenes {
		if record.Name == name {
			return record.ID
		}
	}
	t.Fatalf("scene %q not found in snapshot", name)
	return ""
}

type duplicateModule struct{}

func (duplicateModule) Name() string { return "duplicate" }

func (duplicateModule) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{{
		Spec: campaign.CreateCommandSpec,
		Admission: admission.Rule{
			Authorize: func(caller.Caller, campaign.State) error { return nil },
		},
	}}
}

func (duplicateModule) Events() []event.Spec {
	return []event.Spec{campaign.CreatedEventSpec}
}

func (duplicateModule) Decide(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error) {
	return nil, nil
}

func (duplicateModule) Fold(*campaign.State, event.Envelope) error {
	return nil
}

func defaultCaller() caller.Caller {
	return caller.MustNewSubject("subject-1")
}
