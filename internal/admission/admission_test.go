package admission

import (
	"errors"
	"testing"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
)

func TestNewCatalog(t *testing.T) {
	t.Parallel()

	if _, err := NewCatalog(nil); err == nil {
		t.Fatal("NewCatalog(nil) error = nil, want failure")
	}
	if _, err := NewCatalog(map[command.Type]Rule{"": {Authorize: func(caller.Caller, campaign.State) error { return nil }}}); err == nil {
		t.Fatal("NewCatalog(blank type) error = nil, want failure")
	}
	if _, err := NewCatalog(map[command.Type]Rule{" test.command ": {Authorize: func(caller.Caller, campaign.State) error { return nil }}}); err == nil {
		t.Fatal("NewCatalog(padded type) error = nil, want failure")
	}
	if _, err := NewCatalog(map[command.Type]Rule{"x": {}}); err == nil {
		t.Fatal("NewCatalog(missing authorize) error = nil, want failure")
	}
	if _, err := NewCatalog(map[command.Type]Rule{"x": {Authorize: func(caller.Caller, campaign.State) error { return nil }}}); err != nil {
		t.Fatalf("NewCatalog(unrestricted play states) error = %v", err)
	}
	if _, err := NewCatalog(map[command.Type]Rule{"x": {Authorize: func(caller.Caller, campaign.State) error { return nil }, AllowedPlayStates: []campaign.PlayState{"BROKEN"}}}); err == nil {
		t.Fatal("NewCatalog(invalid play state) error = nil, want failure")
	}
	catalog, err := NewCatalog(testRules())
	if err != nil {
		t.Fatalf("NewCatalog(valid) error = %v", err)
	}
	if catalog == nil {
		t.Fatal("NewCatalog(valid) = nil, want catalog")
	}
}

func TestAdmit(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.Exists = true
	state.PlayState = campaign.PlayStateSetup
	catalog, err := NewCatalog(testRules())
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	if _, err := (*Catalog)(nil).Admit(caller.Caller{}, campaign.State{}, command.Envelope{}); err == nil {
		t.Fatal("nil catalog Admit() error = nil, want failure")
	}
	if _, err := catalog.Admit(caller.MustNewSubject("subject-owner"), state, command.Envelope{Message: testCommand{}}); err == nil {
		t.Fatal("Admit(unknown) error = nil, want failure")
	}

	rule, err := catalog.Admit(caller.MustNewSubject("subject-owner"), state, command.Envelope{CampaignID: "camp-1", Message: modeBoundCommand{}})
	if err != nil {
		t.Fatalf("Admit(modeBound) error = %v", err)
	}
	if rule.SupportsPlanning {
		t.Fatal("modeBound should not opt into planning metadata")
	}
	rule, err = catalog.Admit(caller.MustNewSubject("subject-owner"), state, command.Envelope{CampaignID: "camp-1", Message: plannedCommand{}})
	if err != nil {
		t.Fatalf("Admit(planned) error = %v", err)
	}
	if !rule.SupportsPlanning {
		t.Fatal("planned command should be marked as plannable")
	}
	rule, err = catalog.Admit(caller.MustNewSubject("subject-owner"), state, command.Envelope{CampaignID: "camp-1", Message: liveOnlyCommand{}})
	if err != nil {
		t.Fatalf("Admit(liveOnly) error = %v", err)
	}
	if rule.SupportsPlanning {
		t.Fatal("liveOnly should not be marked as plannable")
	}
	if _, err := catalog.Admit(caller.MustNewSubject("subject-owner"), state, command.Envelope{CampaignID: "camp-1", Message: authzCommand{}}); err != nil {
		t.Fatalf("Admit(authz ok) error = %v", err)
	}

	state.PlayState = campaign.PlayStateActive
	if _, err := catalog.Admit(caller.MustNewSubject("subject-owner"), state, command.Envelope{CampaignID: "camp-1", Message: modeBoundCommand{}}); err == nil {
		t.Fatal("Admit(modeBound in play) error = nil, want failure")
	}

	wantErr := errors.New("deny")
	catalog, err = NewCatalog(map[command.Type]Rule{
		authzCommandType: {
			Authorize: func(caller.Caller, campaign.State) error { return wantErr },
		},
	})
	if err != nil {
		t.Fatalf("NewCatalog(authz error) error = %v", err)
	}
	if _, err := catalog.Admit(caller.MustNewSubject("subject-owner"), state, command.Envelope{CampaignID: "camp-1", Message: authzCommand{}}); !errors.Is(err, wantErr) {
		t.Fatalf("Admit(authz error) error = %v, want %v", err, wantErr)
	}
	if _, ok := (*Catalog)(nil).RuleFor(modeBoundCommandType); ok {
		t.Fatal("nil catalog RuleFor() ok = true, want false")
	}
	catalog, err = NewCatalog(testRules())
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}
	if _, ok := catalog.RuleFor(command.Type(" mode.bound ")); ok {
		t.Fatal("RuleFor(trimmed type) ok = true, want false")
	}
	if _, ok := catalog.RuleFor(command.Type("missing.command")); ok {
		t.Fatal("RuleFor(missing type) ok = true, want false")
	}
}

type testCommand struct{}

func (testCommand) CommandType() command.Type { return "test.command" }

type modeBoundCommand struct{}

func (modeBoundCommand) CommandType() command.Type { return modeBoundCommandType }

type plannedCommand struct{}

func (plannedCommand) CommandType() command.Type { return plannedCommandType }

type liveOnlyCommand struct{}

func (liveOnlyCommand) CommandType() command.Type { return liveOnlyCommandType }

type authzCommand struct{}

func (authzCommand) CommandType() command.Type { return authzCommandType }

const (
	modeBoundCommandType command.Type = "mode.bound"
	plannedCommandType   command.Type = "planned.command"
	liveOnlyCommandType  command.Type = "live.only"
	authzCommandType     command.Type = "authz.command"
)

func testRules() map[command.Type]Rule {
	return map[command.Type]Rule{
		modeBoundCommandType: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup},
		},
		plannedCommandType: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup, campaign.PlayStateActive},
			SupportsPlanning:  true,
		},
		liveOnlyCommandType: {
			Authorize:         func(caller.Caller, campaign.State) error { return nil },
			AllowedPlayStates: []campaign.PlayState{campaign.PlayStateSetup, campaign.PlayStateActive},
		},
		authzCommandType: {
			Authorize: func(caller.Caller, campaign.State) error { return nil },
		},
	}
}
