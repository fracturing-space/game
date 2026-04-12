package memory

import "github.com/fracturing-space/game/internal/service"

// Bundle groups the in-memory implementations of the core storage ports.
type Bundle struct {
	Journal         service.Journal
	ProjectionStore service.ProjectionStore
	ArtifactStore   service.ArtifactStore
}

// NewBundle returns one in-memory bundle for tests and ephemeral runtimes.
func NewBundle() Bundle {
	return Bundle{
		Journal:         NewJournal(),
		ProjectionStore: NewProjectionStore(),
		ArtifactStore:   NewArtifactStore(),
	}
}
