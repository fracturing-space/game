package eventjournal

import (
	"path/filepath"
	"testing"

	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/storage/contracttest"
)

func TestJournalContracts(t *testing.T) {
	contracttest.RunJournal(t, func(t *testing.T) service.Journal {
		t.Helper()

		manifest, err := service.BuildManifest(nil)
		if err != nil {
			t.Fatalf("BuildManifest() error = %v", err)
		}
		store, err := Open(filepath.Join(t.TempDir(), "events.db"), manifest.Events)
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
