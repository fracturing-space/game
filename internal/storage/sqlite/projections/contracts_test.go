package projections

import (
	"path/filepath"
	"testing"

	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/storage/contracttest"
)

func TestProjectionStoreContracts(t *testing.T) {
	contracttest.RunProjectionStore(t, func(t *testing.T) service.ProjectionStore {
		t.Helper()

		store, err := Open(filepath.Join(t.TempDir(), "projections.db"))
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
