package service

import (
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/engine"
	modulecampaign "github.com/fracturing-space/game/internal/modules/campaign"
	modulecharacter "github.com/fracturing-space/game/internal/modules/character"
	moduleparticipant "github.com/fracturing-space/game/internal/modules/participant"
)

func TestRealClockNow(t *testing.T) {
	t.Parallel()

	now := realRecordClock{}.Now()
	if now.IsZero() {
		t.Fatal("Now() should return a non-zero time")
	}
	if got, want := now.Location(), time.UTC; got != want {
		t.Fatalf("location = %v, want %v", got, want)
	}
}

func TestNewUsesDefaults(t *testing.T) {
	t.Parallel()

	svc, err := New(Config{
		Journal:         newTestMemoryStore(),
		ProjectionStore: newTestProjectionStore(),
		ArtifactStore:   newTestArtifactStore(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, ok := svc.recordClock.(realRecordClock); !ok {
		t.Fatalf("clock = %T, want realRecordClock", svc.recordClock)
	}

	customClock := fixedClock{at: serviceTestClockTime}
	custom, err := New(Config{
		Manifest:        mustManifest(t, []engine.Module{modulecampaign.New(), modulecharacter.New(), moduleparticipant.New()}),
		RecordClock:     customClock,
		Journal:         newTestMemoryStore(),
		ProjectionStore: newTestProjectionStore(),
		ArtifactStore:   newTestArtifactStore(),
	})
	if err != nil {
		t.Fatalf("New(custom) error = %v", err)
	}
	if custom.recordClock != customClock {
		t.Fatal("custom record clock was not preserved")
	}

	customJournal := newTestMemoryStore()
	withDependencies, err := New(Config{
		Manifest:        mustManifest(t, []engine.Module{modulecampaign.New(), modulecharacter.New(), moduleparticipant.New()}),
		RecordClock:     customClock,
		Journal:         customJournal,
		ProjectionStore: newTestProjectionStore(),
		ArtifactStore:   newTestArtifactStore(),
	})
	if err != nil {
		t.Fatalf("New(with dependencies) error = %v", err)
	}
	if withDependencies.store != customJournal {
		t.Fatal("custom journal was not preserved")
	}
}

func TestNewRejectsInvalidAdmissionRule(t *testing.T) {
	t.Parallel()

	_, err := BuildManifest([]engine.Module{invalidAdmissionModule{}})
	if err == nil {
		t.Fatal("BuildManifest() error = nil, want invalid admission rule failure")
	}
	if !strings.Contains(err.Error(), "admission rule authorize is required") {
		t.Fatalf("BuildManifest() error = %v, want invalid admission rule failure", err)
	}
}

func TestNewRejectsInvalidModuleConfiguration(t *testing.T) {
	t.Parallel()

	_, err := BuildManifest([]engine.Module{nil})
	if err == nil {
		t.Fatal("BuildManifest() error = nil, want invalid module failure")
	}
}

func TestNewRequiresStoragePorts(t *testing.T) {
	t.Parallel()

	_, err := New(Config{})
	if err == nil {
		t.Fatal("New() error = nil, want missing storage failure")
	}
	if !strings.Contains(err.Error(), "journal is required") {
		t.Fatalf("New() error = %v, want missing journal failure", err)
	}
}

func mustManifest(t *testing.T, modules []engine.Module) *Manifest {
	t.Helper()

	manifest, err := BuildManifest(modules)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	return manifest
}
