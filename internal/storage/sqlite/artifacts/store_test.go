package artifacts

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/service"
)

var fixedRecordTime = time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)

func TestStoreNormalizesAndPersistsArtifacts(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "artifacts.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}

	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: "camp-1",
		Path:       "story.md",
		Content:    "# Harbor",
		UpdatedAt:  fixedRecordTime,
	}); err != nil {
		t.Fatalf("PutArtifact() error = %v", err)
	}
	item, ok, err := store.GetArtifact(context.Background(), "camp-1", "story.md")
	if err != nil {
		t.Fatalf("GetArtifact() error = %v", err)
	}
	if !ok || item.Content != "# Harbor" {
		t.Fatalf("GetArtifact() = (%+v,%t), want stored artifact", item, ok)
	}
	items, err := store.ListArtifacts(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("ListArtifacts() error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("ListArtifacts() len = %d, want %d", got, want)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	store, err = Open(path)
	if err != nil {
		t.Fatalf("Open(second) error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	}()

	if _, _, err = store.GetArtifact(context.Background(), "camp-1", "/story.md"); err == nil {
		t.Fatal("GetArtifact(reloaded non-canonical path) error = nil, want failure")
	}
	item, ok, err = store.GetArtifact(context.Background(), "camp-1", "story.md")
	if err != nil {
		t.Fatalf("GetArtifact(reloaded canonical) error = %v", err)
	}
	if !ok || item.Path != "story.md" {
		t.Fatalf("GetArtifact(reloaded canonical) = (%+v,%t), want stored path", item, ok)
	}
}

func TestStoreHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "artifacts.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.PutArtifact(ctx, service.Artifact{
		CampaignID: "camp-1",
		Path:       "story.md",
		Content:    "# Harbor",
		UpdatedAt:  fixedRecordTime,
	}); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("PutArtifact(canceled) error = %v, want context canceled", err)
	}
	if _, _, err := store.GetArtifact(ctx, "camp-1", "story.md"); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("GetArtifact(canceled) error = %v, want context canceled", err)
	}
}

func TestOpenPropagatesSQLiteOpenAndMigrationFailures(t *testing.T) {
	t.Run("open failure", func(t *testing.T) {
		originalOpenSQLiteDB := openSQLiteDB
		originalApplyMigrations := applyMigrations
		t.Cleanup(func() {
			openSQLiteDB = originalOpenSQLiteDB
			applyMigrations = originalApplyMigrations
		})

		openErr := errors.New("sqlite open failed")
		openSQLiteDB = func(path string) (*sql.DB, error) {
			return nil, openErr
		}

		if _, err := Open(filepath.Join(t.TempDir(), "artifacts.db")); err == nil || !errors.Is(err, openErr) {
			t.Fatalf("Open(open failure) error = %v, want wrapped sqlite open failure", err)
		}
	})

	t.Run("migration failure", func(t *testing.T) {
		originalOpenSQLiteDB := openSQLiteDB
		originalApplyMigrations := applyMigrations
		t.Cleanup(func() {
			openSQLiteDB = originalOpenSQLiteDB
			applyMigrations = originalApplyMigrations
		})

		migrationErr := errors.New("migration failed")
		applyMigrations = func(sqlDB *sql.DB) error {
			return migrationErr
		}

		if _, err := Open(filepath.Join(t.TempDir(), "artifacts.db")); err == nil || !errors.Is(err, migrationErr) {
			t.Fatalf("Open(migration failure) error = %v, want wrapped migration failure", err)
		}
	})
}

func TestStoreNoOpAndMissingArtifactBranches(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "artifacts.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: "   ",
		Path:       "story.md",
		Content:    "ignored",
		UpdatedAt:  fixedRecordTime,
	}); err == nil {
		t.Fatal("PutArtifact(blank campaign) error = nil, want failure")
	}
	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: "camp-1",
		Path:       "   ",
		Content:    "ignored",
		UpdatedAt:  fixedRecordTime,
	}); err == nil {
		t.Fatal("PutArtifact(blank path) error = nil, want failure")
	}

	item, ok, err := store.GetArtifact(context.Background(), "camp-1", "missing.md")
	if err != nil {
		t.Fatalf("GetArtifact(missing) error = %v", err)
	}
	if ok || item != (service.Artifact{}) {
		t.Fatalf("GetArtifact(missing) = (%+v,%t), want zero,false", item, ok)
	}

	items, err := store.ListArtifacts(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("ListArtifacts(empty) error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("ListArtifacts(empty) len = %d, want 0", len(items))
	}
}

func TestStoreRejectsNonCanonicalBoundaryInputs(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "artifacts.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if _, _, err := store.GetArtifact(context.Background(), " camp-1 ", "story.md"); err == nil {
		t.Fatal("GetArtifact(padded campaign) error = nil, want failure")
	}
	if _, _, err := store.GetArtifact(context.Background(), "camp-1", " story.md "); err == nil {
		t.Fatal("GetArtifact(padded path) error = nil, want failure")
	}
	if _, _, err := store.GetArtifact(context.Background(), "camp-1", "/story.md"); err == nil {
		t.Fatal("GetArtifact(leading slash path) error = nil, want failure")
	}
	if _, err := store.ListArtifacts(context.Background(), " camp-1 "); err == nil {
		t.Fatal("ListArtifacts(padded campaign) error = nil, want failure")
	}
	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: " camp-1 ",
		Path:       "story.md",
		UpdatedAt:  fixedRecordTime,
	}); err == nil {
		t.Fatal("PutArtifact(padded campaign) error = nil, want failure")
	}
	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: "camp-1",
		Path:       " story.md ",
		UpdatedAt:  fixedRecordTime,
	}); err == nil {
		t.Fatal("PutArtifact(padded path) error = nil, want failure")
	}
	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: "camp-1",
		Path:       "/story.md",
		UpdatedAt:  fixedRecordTime,
	}); err == nil {
		t.Fatal("PutArtifact(leading slash path) error = nil, want failure")
	}
}

func TestNilStoreCloseIsSafe(t *testing.T) {
	t.Parallel()

	var nilStore *Store
	if err := nilStore.Close(); err != nil {
		t.Fatalf("(*Store)(nil).Close() error = %v, want nil", err)
	}

	store := &Store{}
	if err := store.Close(); err != nil {
		t.Fatalf("(&Store{}).Close() error = %v, want nil", err)
	}
}
