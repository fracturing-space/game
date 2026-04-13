package sqlite

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/service"
	sqliteartifacts "github.com/fracturing-space/game/internal/storage/sqlite/artifacts"
	sqliteeventjournal "github.com/fracturing-space/game/internal/storage/sqlite/eventjournal"
	sqliteprojections "github.com/fracturing-space/game/internal/storage/sqlite/projections"
)

const (
	// DefaultEventsDBPath stores the immutable campaign journal.
	DefaultEventsDBPath = "data/game-events.db"
	// DefaultProjectionsDBPath stores rebuildable campaign read models.
	DefaultProjectionsDBPath = "data/game-projections.db"
	// DefaultArtifactsDBPath stores authored campaign documents.
	DefaultArtifactsDBPath = "data/game-artifacts.db"
)

// Paths configures the durable sqlite-backed store bundle.
type Paths struct {
	EventsDBPath      string
	ProjectionsDBPath string
	ArtifactsDBPath   string
}

// DefaultPaths returns the default on-disk database locations.
func DefaultPaths() Paths {
	return Paths{
		EventsDBPath:      DefaultEventsDBPath,
		ProjectionsDBPath: DefaultProjectionsDBPath,
		ArtifactsDBPath:   DefaultArtifactsDBPath,
	}
}

// Bundle groups the sqlite implementations of the core storage ports.
type Bundle struct {
	Journal         service.Journal
	ProjectionStore service.ProjectionStore
	ArtifactStore   service.ArtifactStore

	closeFn func() error
}

type journalStore interface {
	service.Journal
	Close() error
}

type projectionStore interface {
	service.ProjectionStore
	Close() error
}

type artifactStore interface {
	service.ArtifactStore
	Close() error
}

var (
	openJournalStore = func(path string, catalog *event.Catalog) (journalStore, error) {
		return sqliteeventjournal.Open(path, catalog)
	}
	openProjectionStore = func(path string) (projectionStore, error) {
		return sqliteprojections.Open(path)
	}
	openArtifactStore = func(path string) (artifactStore, error) {
		return sqliteartifacts.Open(path)
	}
)

// Open opens the sqlite-backed journal, projection, and artifact stores.
func Open(manifest *service.Manifest, paths Paths) (*Bundle, error) {
	if manifest == nil || manifest.Events == nil {
		return nil, fmt.Errorf("service manifest with event catalog is required")
	}
	paths = normalizePaths(paths)

	journal, err := openJournalStore(paths.EventsDBPath, manifest.Events)
	if err != nil {
		return nil, err
	}
	projectionStore, err := openProjectionStore(paths.ProjectionsDBPath)
	if err != nil {
		_ = journal.Close()
		return nil, err
	}
	artifactStore, err := openArtifactStore(paths.ArtifactsDBPath)
	if err != nil {
		_ = projectionStore.Close()
		_ = journal.Close()
		return nil, err
	}

	return &Bundle{
		Journal:         journal,
		ProjectionStore: projectionStore,
		ArtifactStore:   artifactStore,
		closeFn: func() error {
			return errors.Join(
				artifactStore.Close(),
				projectionStore.Close(),
				journal.Close(),
			)
		},
	}, nil
}

// Close closes every store in the bundle.
func (b *Bundle) Close() error {
	if b == nil || b.closeFn == nil {
		return nil
	}
	return b.closeFn()
}

func normalizePaths(paths Paths) Paths {
	defaults := DefaultPaths()
	if strings.TrimSpace(paths.EventsDBPath) == "" {
		paths.EventsDBPath = defaults.EventsDBPath
	}
	if strings.TrimSpace(paths.ProjectionsDBPath) == "" {
		paths.ProjectionsDBPath = defaults.ProjectionsDBPath
	}
	if strings.TrimSpace(paths.ArtifactsDBPath) == "" {
		paths.ArtifactsDBPath = defaults.ArtifactsDBPath
	}
	return paths
}
