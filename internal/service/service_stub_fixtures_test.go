package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
	modulecampaign "github.com/fracturing-space/game/internal/modules/campaign"
	moduleparticipant "github.com/fracturing-space/game/internal/modules/participant"
	modulesession "github.com/fracturing-space/game/internal/modules/session"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/session"
)

type stubCampaignFixture struct {
	Service    *Service
	CampaignID string
}

func newStubService(t *testing.T, module stubServiceModule) *Service {
	t.Helper()
	return newStubServiceWithLogger(t, module, nil)
}

func newStubServiceWithLogger(t *testing.T, module stubServiceModule, logger *slog.Logger) *Service {
	t.Helper()

	manifest, err := BuildManifest([]engine.Module{modulecampaign.New(), moduleparticipant.New(), modulesession.New(), module.withDefaults()})
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	svc, err := New(Config{
		Manifest:        manifest,
		IDs:             newSequentialIDAllocator(),
		RecordClock:     fixedClock{at: serviceTestClockTime},
		Journal:         newTestMemoryStore(),
		ProjectionStore: newTestProjectionStore(),
		ArtifactStore:   newTestArtifactStore(),
		Logger:          logger,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return svc
}

func newGMEnabledStubCampaignFixture(t *testing.T, module stubServiceModule) stubCampaignFixture {
	t.Helper()

	fixture := stubCampaignFixture{
		Service:    newStubService(t, module),
		CampaignID: "camp-1",
	}
	seedGMEnabledCampaign(t, fixture.Service, fixture.CampaignID)
	return fixture
}

func mustTestEventEnvelope(t *testing.T, campaignID, name string) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(testEventSpec, campaignID, testEvent{Name: name})
	if err != nil {
		t.Fatalf("NewEnvelope() error = %v", err)
	}
	return envelope
}

type testNewCommand struct {
	Name string
}

func (testNewCommand) CommandType() command.Type { return "test.new" }

type testCampaignCommand struct {
	Name string
}

func (testCampaignCommand) CommandType() command.Type { return "test.campaign" }

type testLiveCommand struct {
	Name string
}

func (testLiveCommand) CommandType() command.Type { return "test.live" }

type testEvent struct {
	Name string
}

func (testEvent) EventType() event.Type { return "test.event" }

type unknownTestEvent struct{}

func (unknownTestEvent) EventType() event.Type { return "test.unknown" }

var testNewCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[testNewCommand]{
	Message:   testNewCommand{},
	Scope:     command.ScopeNewCampaign,
	Normalize: normalizeTestNewCommand,
	Validate:  validateTestNewCommand,
})
var testCampaignCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[testCampaignCommand]{
	Message:   testCampaignCommand{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeTestCampaignCommand,
	Validate:  validateTestCampaignCommand,
})
var testLiveCommandSpec = command.NewCoreSpec(command.CoreSpecArgs[testLiveCommand]{
	Message:   testLiveCommand{},
	Scope:     command.ScopeCampaign,
	Normalize: normalizeTestLiveCommand,
	Validate:  validateTestLiveCommand,
})
var testEventSpec = event.NewCoreSpec(testEvent{}, normalizeTestEvent, validateTestEventName)

func validateTestNewCommand(message testNewCommand) error {
	if message.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func validateTestCampaignCommand(message testCampaignCommand) error {
	if message.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func validateTestLiveCommand(message testLiveCommand) error {
	if message.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func validateTestEventName(message testEvent) error {
	if message.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func normalizeTestNewCommand(message testNewCommand) testNewCommand {
	message.Name = strings.TrimSpace(message.Name)
	return message
}

func normalizeTestCampaignCommand(message testCampaignCommand) testCampaignCommand {
	message.Name = strings.TrimSpace(message.Name)
	return message
}

func normalizeTestLiveCommand(message testLiveCommand) testLiveCommand {
	message.Name = strings.TrimSpace(message.Name)
	return message
}

func normalizeTestEvent(message testEvent) testEvent {
	message.Name = strings.TrimSpace(message.Name)
	return message
}

type stubServiceModule struct {
	decide func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error)
	fold   func(*campaign.State, event.Envelope) error
	rules  map[command.Type]admission.Rule
}

func (m stubServiceModule) withDefaults() stubServiceModule {
	if m.decide == nil {
		m.decide = defaultStubDecide
	}
	if m.fold == nil {
		m.fold = defaultStubFold
	}
	if m.rules == nil {
		m.rules = testServiceAdmissionRules()
	}
	return m
}

func (stubServiceModule) Name() string { return "stub.service" }

func (m stubServiceModule) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{
		{Spec: testNewCommandSpec, Admission: m.rules[testNewCommandSpec.Definition().Type]},
		{Spec: testCampaignCommandSpec, Admission: m.rules[testCampaignCommandSpec.Definition().Type]},
		{Spec: testLiveCommandSpec, Admission: m.rules[testLiveCommandSpec.Definition().Type]},
	}
}

func (stubServiceModule) Events() []event.Spec {
	return []event.Spec{testEventSpec}
}

func (m stubServiceModule) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	return m.decide(state, act, envelope, ids)
}

func (m stubServiceModule) Fold(state *campaign.State, envelope event.Envelope) error {
	return m.fold(state, envelope)
}

func defaultStubDecide(state campaign.State, _ caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	switch message := envelope.Message.(type) {
	case testNewCommand:
		campaignID, err := ids("camp")
		if err != nil {
			return nil, err
		}
		next, err := event.NewEnvelope(testEventSpec, campaignID, testEvent(message))
		if err != nil {
			return nil, err
		}
		return []event.Envelope{next}, nil
	case testCampaignCommand:
		if !state.Exists && state.CampaignID == "" {
			return nil, fmt.Errorf("campaign state missing")
		}
		next, err := event.NewEnvelope(testEventSpec, envelope.CampaignID, testEvent(message))
		if err != nil {
			return nil, err
		}
		return []event.Envelope{next}, nil
	case testLiveCommand:
		if !state.Exists && state.CampaignID == "" {
			return nil, fmt.Errorf("campaign state missing")
		}
		next, err := event.NewEnvelope(testEventSpec, envelope.CampaignID, testEvent(message))
		if err != nil {
			return nil, err
		}
		return []event.Envelope{next}, nil
	default:
		return nil, fmt.Errorf("unsupported command %s", envelope.Type())
	}
}

func testServiceAdmissionRules() map[command.Type]admission.Rule {
	return map[command.Type]admission.Rule{
		campaign.CommandTypeCreate: {
			Authorize: func(caller.Caller, campaign.State) error { return nil },
		},
		character.CommandTypeCreate: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
		},
		campaign.CommandTypeAIBind: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup, campaign.PlayStateActive},
		},
		campaign.CommandTypeAIUnbind: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup, campaign.PlayStateActive},
		},
		campaign.CommandTypePlayPause: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive},
		},
		campaign.CommandTypePlayResume: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStatePaused},
		},
		session.CommandTypeStart: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
		},
		session.CommandTypeEnd: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateActive, campaign.PlayStatePaused},
		},
		participant.CommandTypeJoin: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
		},
		testNewCommandSpec.Definition().Type: {
			Authorize: func(caller.Caller, campaign.State) error { return nil },
		},
		testCampaignCommandSpec.Definition().Type: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup, campaign.PlayStateActive},
			SupportsPlanning:  true,
		},
		testLiveCommandSpec.Definition().Type: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup, campaign.PlayStateActive},
		},
	}
}

func defaultStubFold(state *campaign.State, envelope event.Envelope) error {
	message, err := event.MessageAs[testEvent](envelope)
	if err != nil {
		return err
	}
	state.Exists = true
	state.CampaignID = envelope.CampaignID
	state.Name = message.Name
	state.PlayState = campaign.PlayStateSetup
	return nil
}

type invalidAdmissionModule struct{}

func (invalidAdmissionModule) Name() string { return "invalid.admission" }

func (invalidAdmissionModule) Commands() []engine.CommandRegistration {
	return []engine.CommandRegistration{{
		Spec: testNewCommandSpec,
	}}
}

func (invalidAdmissionModule) Events() []event.Spec { return []event.Spec{testEventSpec} }

func (invalidAdmissionModule) Decide(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error) {
	return nil, nil
}

func (invalidAdmissionModule) Fold(*campaign.State, event.Envelope) error { return nil }

func seedAuthorizedCampaign(t *testing.T, svc *Service, campaignID string, act caller.Caller) {
	t.Helper()

	if _, err := svc.store.AppendCommits(context.Background(), campaignID, []PreparedCommit{
		{Events: []event.Envelope{
			mustTestEventEnvelope(t, campaignID, "start"),
			mustParticipantJoinedEnvelope(t, campaignID, "part-1", "Owner", participant.AccessOwner, act.SubjectID),
		}},
	}, func() time.Time {
		return serviceTestClockTime
	}); err != nil {
		t.Fatalf("AppendCommits(seedAuthorizedCampaign) error = %v", err)
	}
}

func seedGMEnabledCampaign(t *testing.T, svc *Service, campaignID string) {
	t.Helper()

	if _, err := svc.store.AppendCommits(context.Background(), campaignID, []PreparedCommit{
		{Events: []event.Envelope{
			mustTestEventEnvelope(t, campaignID, "start"),
			mustParticipantJoinedEnvelope(t, campaignID, "owner-1", "Owner", participant.AccessOwner, defaultCaller().SubjectID),
			mustCampaignAIBoundEnvelope(t, campaignID, "agent-1"),
			mustSessionStartedEnvelope(t, campaignID, "sess-1", "Session 1"),
			mustCampaignPlayBeganEnvelope(t, campaignID, "sess-1", "scene-1"),
		}},
	}, func() time.Time {
		return serviceTestClockTime
	}); err != nil {
		t.Fatalf("AppendCommits(seedGMEnabledCampaign) error = %v", err)
	}
}

func mustSessionStartedEnvelope(t *testing.T, campaignID, sessionID, name string) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(session.StartedEventSpec, campaignID, session.Started{
		SessionID: sessionID,
		Name:      name,
	})
	if err != nil {
		t.Fatalf("NewEnvelope(session started) error = %v", err)
	}
	return envelope
}

func mustParticipantJoinedEnvelope(t *testing.T, campaignID, participantID, name string, access participant.Access, subjectID string) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(participant.JoinedEventSpec, campaignID, participant.Joined{
		ParticipantID: participantID,
		Name:          name,
		Access:        access,
		SubjectID:     subjectID,
	})
	if err != nil {
		t.Fatalf("NewEnvelope(joined) error = %v", err)
	}
	return envelope
}

func mustCampaignAIBoundEnvelope(t *testing.T, campaignID, aiAgentID string) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(campaign.AIBoundEventSpec, campaignID, campaign.AIBound{AIAgentID: aiAgentID})
	if err != nil {
		t.Fatalf("NewEnvelope(ai bound) error = %v", err)
	}
	return envelope
}

func mustCampaignPlayBeganEnvelope(t *testing.T, campaignID, sessionID, sceneID string) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(campaign.PlayBeganEventSpec, campaignID, campaign.PlayBegan{
		SessionID: sessionID,
		SceneID:   sceneID,
	})
	if err != nil {
		t.Fatalf("NewEnvelope(play began) error: %v", err)
	}
	return envelope
}
