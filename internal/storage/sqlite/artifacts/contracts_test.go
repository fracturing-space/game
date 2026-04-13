package artifacts

import (
	"path/filepath"
	"testing"

	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/storage/contracttest"
)

func TestArtifactStoreContracts(t *testing.T) {
	contracttest.RunArtifactStore(t, func(t *testing.T) service.ArtifactStore {
		t.Helper()

		store, err := Open(filepath.Join(t.TempDir(), "artifacts.db"))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		t.Cleanup(func() {
			if err := store.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})
		return store
	})
}
