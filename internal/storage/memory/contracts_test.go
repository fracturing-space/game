package memory

import (
	"testing"

	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/storage/contracttest"
)

func TestJournalContracts(t *testing.T) {
	contracttest.RunJournal(t, func(*testing.T) service.Journal {
		return NewJournal()
	})
}

func TestProjectionStoreContracts(t *testing.T) {
	contracttest.RunProjectionStore(t, func(*testing.T) service.ProjectionStore {
		return NewProjectionStore()
	})
}

func TestArtifactStoreContracts(t *testing.T) {
	contracttest.RunArtifactStore(t, func(*testing.T) service.ArtifactStore {
		return NewArtifactStore()
	})
}
