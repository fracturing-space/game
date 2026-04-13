package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/fracturing-space/game/internal/admission"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/engine"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/modules/campaign"
	"github.com/fracturing-space/game/internal/modules/character"
	"github.com/fracturing-space/game/internal/modules/participant"
	"github.com/fracturing-space/game/internal/modules/scene"
	"github.com/fracturing-space/game/internal/modules/session"
)

// DefaultCoreModules returns the standard core module set.
func DefaultCoreModules() []engine.Module {
	return []engine.Module{
		campaign.New(),
		character.New(),
		participant.New(),
		scene.New(),
		session.New(),
	}
}

// DefaultSystemModules returns built-in system modules.
//
// The current service ships only core modules, but this hook keeps system-owned
// commands and events as an explicit extension seam for future rules modules.
func DefaultSystemModules() []engine.Module {
	return nil
}

// DefaultModules returns the standard game module set.
func DefaultModules() []engine.Module {
	return append(DefaultCoreModules(), DefaultSystemModules()...)
}

// Manifest is the built domain runtime used by the core service.
type Manifest struct {
	Registry  *engine.Registry
	Commands  *command.Catalog
	Events    *event.Catalog
	Admission *admission.Catalog
}

// BuildManifest validates the provided modules and builds the domain runtime
// once for service and adapter wiring.
func BuildManifest(modules []engine.Module) (*Manifest, error) {
	if len(modules) == 0 {
		modules = DefaultModules()
	}
	artifacts, err := engine.Build(modules...)
	if err != nil {
		return nil, err
	}
	return &Manifest{
		Registry:  artifacts.Registry,
		Commands:  artifacts.Commands,
		Events:    artifacts.Events,
		Admission: artifacts.Admission,
	}, nil
}

// Config wires one service instance.
type Config struct {
	Manifest        *Manifest
	IDs             IDAllocator
	RecordClock     RecordClock
	Journal         Journal
	ProjectionStore ProjectionStore
	ArtifactStore   ArtifactStore
	RuntimeIdleTTL  time.Duration
	Logger          *slog.Logger
}

// Service executes commands against the configured journal and read-model
// stores.
type Service struct {
	recordClock RecordClock
	registry    *engine.Registry
	commands    *command.Catalog
	events      *event.Catalog
	admission   *admission.Catalog
	store       Journal
	projections ProjectionStore
	artifacts   ArtifactStore
	slots       *campaignSlotRegistry
	plans       *planStore
	ids         IDAllocator
	logger      *slog.Logger
	runtimeTTL  time.Duration
	slotIdleTTL time.Duration
	planTTL     time.Duration
}

// New constructs one replay-first core service.
func New(cfg Config) (*Service, error) {
	if cfg.RecordClock == nil {
		cfg.RecordClock = realRecordClock{}
	}
	if cfg.IDs == nil {
		cfg.IDs = NewOpaqueIDAllocator()
	}
	if cfg.Journal == nil {
		return nil, fmt.Errorf("journal is required")
	}
	if cfg.ProjectionStore == nil {
		return nil, fmt.Errorf("projection store is required")
	}
	if cfg.ArtifactStore == nil {
		return nil, fmt.Errorf("artifact store is required")
	}
	if cfg.RuntimeIdleTTL <= 0 {
		cfg.RuntimeIdleTTL = 5 * time.Minute
	}

	if cfg.Manifest == nil {
		manifest, err := BuildManifest(nil)
		if err != nil {
			return nil, err
		}
		cfg.Manifest = manifest
	}
	if cfg.Manifest.Registry == nil || cfg.Manifest.Commands == nil || cfg.Manifest.Events == nil || cfg.Manifest.Admission == nil {
		return nil, fmt.Errorf("service manifest is incomplete")
	}
	return &Service{
		recordClock: cfg.RecordClock,
		registry:    cfg.Manifest.Registry,
		commands:    cfg.Manifest.Commands,
		events:      cfg.Manifest.Events,
		admission:   cfg.Manifest.Admission,
		store:       cfg.Journal,
		projections: cfg.ProjectionStore,
		artifacts:   cfg.ArtifactStore,
		slots:       newCampaignSlotRegistry(),
		plans: &planStore{
			items:      make(map[string]preparedPlan),
			byCampaign: make(map[string]string),
		},
		ids:         cfg.IDs,
		logger:      withServiceLogger(cfg.Logger),
		runtimeTTL:  cfg.RuntimeIdleTTL,
		slotIdleTTL: 2 * cfg.RuntimeIdleTTL,
		planTTL:     5 * time.Minute,
	}, nil
}
