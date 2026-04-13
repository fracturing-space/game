package engine

import (
	"fmt"

	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
)

// CommandRegistration binds one command spec to its admission rule.
type CommandRegistration struct {
	Spec      command.Spec
	Admission admission.Rule
}

// Module is the extension seam for core or system-owned command/event behavior.
type Module interface {
	Name() string
	Commands() []CommandRegistration
	Events() []event.Spec
	Decide(campaign.State, caller.Caller, command.Envelope, func(string) (string, error)) ([]event.Envelope, error)
	Fold(*campaign.State, event.Envelope) error
}

// Registry routes commands and events to their owning modules.
type Registry struct {
	commandModules map[command.Type]Module
	eventModules   map[event.Type]Module
}

// Artifacts contains the assembled engine components used by the service.
type Artifacts struct {
	Registry  *Registry
	Commands  *command.Catalog
	Events    *event.Catalog
	Admission *admission.Catalog
}

// Build assembles the engine registry and authoritative catalogs for the service.
func Build(modules ...Module) (Artifacts, error) {
	if len(modules) == 0 {
		return Artifacts{}, fmt.Errorf("at least one module is required")
	}

	registry := &Registry{
		commandModules: make(map[command.Type]Module),
		eventModules:   make(map[event.Type]Module),
	}
	commandSpecs := make([]command.Spec, 0)
	eventSpecs := make([]event.Spec, 0)
	admissionRules := make(map[command.Type]admission.Rule)

	for _, module := range modules {
		if module == nil {
			return Artifacts{}, fmt.Errorf("module is required")
		}
		for _, registration := range module.Commands() {
			spec := registration.Spec
			definition := spec.Definition()
			if _, exists := registry.commandModules[definition.Type]; exists {
				return Artifacts{}, fmt.Errorf("command type already routed: %s", definition.Type)
			}
			registry.commandModules[definition.Type] = module
			commandSpecs = append(commandSpecs, spec)
			admissionRules[definition.Type] = registration.Admission
		}
		for _, spec := range module.Events() {
			definition := spec.Definition()
			if _, exists := registry.eventModules[definition.Type]; exists {
				return Artifacts{}, fmt.Errorf("event type already routed: %s", definition.Type)
			}
			registry.eventModules[definition.Type] = module
			eventSpecs = append(eventSpecs, spec)
		}
	}

	commandCatalog, err := command.NewCatalog(commandSpecs...)
	if err != nil {
		return Artifacts{}, err
	}
	eventCatalog, err := event.NewCatalog(eventSpecs...)
	if err != nil {
		return Artifacts{}, err
	}
	admissionCatalog, err := admission.NewCatalog(admissionRules)
	if err != nil {
		return Artifacts{}, err
	}
	return Artifacts{
		Registry:  registry,
		Commands:  commandCatalog,
		Events:    eventCatalog,
		Admission: admissionCatalog,
	}, nil
}

// Decide routes one validated command to its owning module.
func (r *Registry) Decide(state campaign.State, act caller.Caller, envelope command.Envelope, ids func(string) (string, error)) ([]event.Envelope, error) {
	module, ok := r.commandModules[envelope.Type()]
	if !ok {
		return nil, fmt.Errorf("command type is not routed: %s", envelope.Type())
	}
	return module.Decide(state, act, envelope, ids)
}

// Fold routes one validated event to its owning module.
func (r *Registry) Fold(state *campaign.State, envelope event.Envelope) error {
	module, ok := r.eventModules[envelope.Type()]
	if !ok {
		return fmt.Errorf("event type is not routed: %s", envelope.Type())
	}
	return module.Fold(state, envelope)
}
