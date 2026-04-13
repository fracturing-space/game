package admission

import (
	"fmt"
	"slices"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
)

// Rule describes how one command is admitted before domain decision logic.
type Rule struct {
	Authorize         func(caller.Caller, campaign.State) error
	AllowedPlayStates []campaign.PlayState
	SupportsPlanning  bool
}

// Catalog validates commands against caller authz and play-state requirements.
type Catalog struct {
	rules map[command.Type]Rule
}

// NewCatalog constructs a validated admission catalog.
func NewCatalog(rules map[command.Type]Rule) (*Catalog, error) {
	if len(rules) == 0 {
		return nil, fmt.Errorf("admission rules are required")
	}
	next := make(map[command.Type]Rule, len(rules))
	for typ, rule := range rules {
		if typ == "" {
			return nil, fmt.Errorf("admission command type is required")
		}
		if !canonical.IsExact(string(typ)) {
			return nil, fmt.Errorf("admission command type must not contain surrounding whitespace: %s", typ)
		}
		if rule.Authorize == nil {
			return nil, fmt.Errorf("admission rule authorize is required: %s", typ)
		}
		for _, playState := range rule.AllowedPlayStates {
			if !playState.Valid() {
				return nil, fmt.Errorf("admission play state is invalid for %s: %s", typ, playState)
			}
		}
		next[typ] = rule
	}
	return &Catalog{rules: next}, nil
}

// RuleFor returns the registered rule for one normalized command type.
func (c *Catalog) RuleFor(typ command.Type) (Rule, bool) {
	if c == nil {
		return Rule{}, false
	}
	rule, ok := c.rules[typ]
	return rule, ok
}

// Admit validates one command against the caller and current campaign state.
func (c *Catalog) Admit(act caller.Caller, state campaign.State, envelope command.Envelope) (Rule, error) {
	if c == nil {
		return Rule{}, fmt.Errorf("admission catalog is required")
	}
	rule, ok := c.rules[envelope.Type()]
	if !ok {
		return Rule{}, fmt.Errorf("admission rule is not registered: %s", envelope.Type())
	}
	if err := rule.Authorize(act, state); err != nil {
		return Rule{}, err
	}
	if state.Exists && len(rule.AllowedPlayStates) != 0 {
		allowed := slices.Contains(rule.AllowedPlayStates, state.PlayState)
		if !allowed {
			return Rule{}, errs.FailedPreconditionf("command %s is not allowed in %s play state", envelope.Type(), state.PlayState)
		}
	}
	return rule, nil
}
