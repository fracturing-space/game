package engine

import (
	"errors"
	"reflect"
	"testing"

	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	t.Run("no modules", func(t *testing.T) {
		if _, err := Build(); err == nil {
			t.Fatal("Build() error = nil, want failure")
		}
	})

	t.Run("nil module", func(t *testing.T) {
		if _, err := Build(nil); err == nil {
			t.Fatal("Build() error = nil, want failure")
		}
	})

	t.Run("duplicate command route", func(t *testing.T) {
		module := testModule{name: "one", commands: []CommandRegistration{testCommandRegistration(command.NewCoreSpec(command.CoreSpecArgs[testCommand]{Message: testCommand{}, Scope: command.ScopeCampaign}))}}
		if _, err := Build(module, module); err == nil {
			t.Fatal("Build() error = nil, want failure")
		}
	})

	t.Run("duplicate event route", func(t *testing.T) {
		moduleA := testModule{name: "one", commands: []CommandRegistration{testCommandRegistration(command.NewCoreSpec(command.CoreSpecArgs[testCommand]{Message: testCommand{}, Scope: command.ScopeCampaign}))}, events: []event.Spec{event.NewCoreSpec(testEvent{}, event.Identity[testEvent], nil)}}
		moduleB := testModule{name: "two", commands: []CommandRegistration{testCommandRegistration(command.NewCoreSpec(command.CoreSpecArgs[otherCommand]{Message: otherCommand{}, Scope: command.ScopeCampaign}))}, events: []event.Spec{event.NewCoreSpec(testEvent{}, event.Identity[testEvent], nil)}}
		if _, err := Build(moduleA, moduleB); err == nil {
			t.Fatal("Build() error = nil, want failure")
		}
	})

	t.Run("invalid command catalog", func(t *testing.T) {
		module := testModule{name: "broken", commands: []CommandRegistration{testCommandRegistration(commandStubSpec{def: command.Definition{Type: "bad.command", Owner: command.Owner("wat"), Scope: command.ScopeCampaign, MessageType: reflect.TypeFor[testCommand]()}})}}
		if _, err := Build(module); err == nil {
			t.Fatal("Build() error = nil, want failure")
		}
	})

	t.Run("invalid admission catalog", func(t *testing.T) {
		module := testModule{
			name:     "broken",
			commands: []CommandRegistration{{Spec: command.NewCoreSpec(command.CoreSpecArgs[testCommand]{Message: testCommand{}, Scope: command.ScopeCampaign})}},
		}
		if _, err := Build(module); err == nil {
			t.Fatal("Build() error = nil, want failure")
		}
	})

	t.Run("invalid event catalog", func(t *testing.T) {
		module := testModule{
			name:     "broken",
			commands: []CommandRegistration{testCommandRegistration(command.NewCoreSpec(command.CoreSpecArgs[testCommand]{Message: testCommand{}, Scope: command.ScopeCampaign}))},
			events:   []event.Spec{eventStubSpec{def: event.Definition{Type: "bad.event", Owner: event.Owner("wat"), MessageType: reflect.TypeFor[testEvent]()}}},
		}
		if _, err := Build(module); err == nil {
			t.Fatal("Build() error = nil, want failure")
		}
	})

	t.Run("success", func(t *testing.T) {
		module := testModule{
			name:     "ok",
			commands: []CommandRegistration{testCommandRegistration(command.NewCoreSpec(command.CoreSpecArgs[testCommand]{Message: testCommand{}, Scope: command.ScopeCampaign}))},
			events:   []event.Spec{event.NewCoreSpec(testEvent{}, event.Identity[testEvent], nil)},
		}
		artifacts, err := Build(module)
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if artifacts.Registry == nil || artifacts.Commands == nil || artifacts.Events == nil || artifacts.Admission == nil {
			t.Fatal("Build() should return assembled artifacts")
		}
	})
}

func TestRegistryDecideAndFold(t *testing.T) {
	t.Parallel()

	module := &testModule{
		name:     "ok",
		commands: []CommandRegistration{testCommandRegistration(command.NewCoreSpec(command.CoreSpecArgs[testCommand]{Message: testCommand{}, Scope: command.ScopeCampaign}))},
		events:   []event.Spec{event.NewCoreSpec(testEvent{}, event.Identity[testEvent], nil)},
		decide: func(state campaign.State, _ caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
			return []event.Envelope{{
				CampaignID: envelope.CampaignID,
				Message:    testEvent{},
			}}, nil
		},
		fold: func(state *campaign.State, envelope event.Envelope) error {
			state.Exists = true
			state.CampaignID = envelope.CampaignID
			return nil
		},
	}

	artifacts, err := Build(module)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if _, err := artifacts.Registry.Decide(campaign.NewState(), caller.MustNewSubject("subject-1"), command.Envelope{Message: otherCommand{}}, nil); err == nil {
		t.Fatal("Decide() should reject unrouted command")
	}
	state := campaign.NewState()
	if err := artifacts.Registry.Fold(&state, event.Envelope{Message: otherEvent{}}); err == nil {
		t.Fatal("Fold() should reject unrouted event")
	}

	envelopes, err := artifacts.Registry.Decide(campaign.NewState(), caller.MustNewSubject("subject-1"), command.Envelope{CampaignID: "camp-1", Message: testCommand{}}, nil)
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if got, want := len(envelopes), 1; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
	state = campaign.NewState()
	if err := artifacts.Registry.Fold(&state, envelopes[0]); err != nil {
		t.Fatalf("Fold() error = %v", err)
	}
	if !state.Exists || state.CampaignID != "camp-1" {
		t.Fatalf("state = %#v, want folded campaign id", state)
	}
}

type testModule struct {
	name     string
	commands []CommandRegistration
	events   []event.Spec
	decide   func(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error)
	fold     func(*campaign.State, event.Envelope) error
}

func (m testModule) Name() string                    { return m.name }
func (m testModule) Commands() []CommandRegistration { return m.commands }
func (m testModule) Events() []event.Spec            { return m.events }
func (m testModule) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	if m.decide == nil {
		return nil, errors.New("no decide")
	}
	return m.decide(state, act, envelope, ids)
}
func (m testModule) Fold(state *campaign.State, envelope event.Envelope) error {
	if m.fold == nil {
		return errors.New("no fold")
	}
	return m.fold(state, envelope)
}

type testCommand struct{}

func (testCommand) CommandType() command.Type { return "test.command" }

type otherCommand struct{}

func (otherCommand) CommandType() command.Type { return "other.command" }

type testEvent struct{}

func (testEvent) EventType() event.Type { return "test.event" }

type otherEvent struct{}

func (otherEvent) EventType() event.Type { return "other.event" }

type commandStubSpec struct {
	def command.Definition
	err error
}

func (s commandStubSpec) Definition() command.Definition { return s.def }
func (s commandStubSpec) NormalizeMessage(message command.Message) (command.Message, error) {
	return message, nil
}
func (s commandStubSpec) ValidateMessage(command.Message) error { return s.err }

type eventStubSpec struct {
	def event.Definition
	err error
}

func (s eventStubSpec) Definition() event.Definition { return s.def }
func (s eventStubSpec) NormalizeMessage(message event.Message) (event.Message, error) {
	return message, nil
}
func (s eventStubSpec) ValidateMessage(event.Message) error { return s.err }

func testCommandRegistration(spec command.Spec) CommandRegistration {
	return CommandRegistration{
		Spec: spec,
		Admission: admission.Rule{
			Authorize: func(caller.Caller, campaign.State) error { return nil },
		},
	}
}
