package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/service"
)

func TestDefaultAndNormalizedPaths(t *testing.T) {
	t.Parallel()

	defaults := DefaultPaths()
	if got, want := defaults.EventsDBPath, DefaultEventsDBPath; got != want {
		t.Fatalf("events db path = %q, want %q", got, want)
	}
	if got, want := defaults.ProjectionsDBPath, DefaultProjectionsDBPath; got != want {
		t.Fatalf("projections db path = %q, want %q", got, want)
	}
	if got, want := defaults.ArtifactsDBPath, DefaultArtifactsDBPath; got != want {
		t.Fatalf("artifacts db path = %q, want %q", got, want)
	}

	normalized := normalizePaths(Paths{})
	if normalized != defaults {
		t.Fatalf("normalizePaths(Paths{}) = %+v, want %+v", normalized, defaults)
	}
}

func TestOpenRequiresManifestAndCloseIsNilSafe(t *testing.T) {
	t.Parallel()

	var nilBundle *Bundle
	if err := nilBundle.Close(); err != nil {
		t.Fatalf("(*Bundle)(nil).Close() error = %v", err)
	}

	if _, err := Open(nil, Paths{}); err == nil || !strings.Contains(err.Error(), "event catalog is required") {
		t.Fatalf("Open(nil) error = %v, want missing manifest failure", err)
	}

	if _, err := Open(&service.Manifest{}, Paths{}); err == nil || !strings.Contains(err.Error(), "event catalog is required") {
		t.Fatalf("Open(empty manifest) error = %v, want missing event catalog failure", err)
	}
}

func TestOpenClosesEarlierStoresWhenLaterOpenFails(t *testing.T) {
	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	originalOpenJournal := openJournalStore
	originalOpenProjection := openProjectionStore
	originalOpenArtifact := openArtifactStore
	t.Cleanup(func() {
		openJournalStore = originalOpenJournal
		openProjectionStore = originalOpenProjection
		openArtifactStore = originalOpenArtifact
	})

	t.Run("projection open failure closes journal", func(t *testing.T) {
		journal := &fakeJournalStore{}
		openJournalStore = func(path string, catalog *event.Catalog) (journalStore, error) {
			return journal, nil
		}
		openProjectionStore = func(path string) (projectionStore, error) {
			return nil, errors.New("projection open failed")
		}
		openArtifactStore = func(path string) (artifactStore, error) {
			t.Fatal("artifact store should not open after projection failure")
			return nil, nil
		}

		if _, err := Open(manifest, Paths{}); err == nil || !strings.Contains(err.Error(), "projection open failed") {
			t.Fatalf("Open() error = %v, want projection open failure", err)
		}
		if !journal.closed {
			t.Fatal("journal.Close() should run after projection open failure")
		}
	})

	t.Run("artifact open failure closes projection and journal", func(t *testing.T) {
		journal := &fakeJournalStore{}
		projections := &fakeProjectionStore{}
		openJournalStore = func(path string, catalog *event.Catalog) (journalStore, error) {
			return journal, nil
		}
		openProjectionStore = func(path string) (projectionStore, error) {
			return projections, nil
		}
		openArtifactStore = func(path string) (artifactStore, error) {
			return nil, errors.New("artifact open failed")
		}

		if _, err := Open(manifest, Paths{}); err == nil || !strings.Contains(err.Error(), "artifact open failed") {
			t.Fatalf("Open() error = %v, want artifact open failure", err)
		}
		if !journal.closed {
			t.Fatal("journal.Close() should run after artifact open failure")
		}
		if !projections.closed {
			t.Fatal("projection.Close() should run after artifact open failure")
		}
	})
}

func TestBundleCloseReturnsJoinedCloseErrors(t *testing.T) {
	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	errJournal := errors.New("journal close failed")
	errProjection := errors.New("projection close failed")
	errArtifact := errors.New("artifact close failed")
	journal := &fakeJournalStore{closeErr: errJournal}
	projections := &fakeProjectionStore{closeErr: errProjection}
	artifacts := &fakeArtifactStore{closeErr: errArtifact}

	originalOpenJournal := openJournalStore
	originalOpenProjection := openProjectionStore
	originalOpenArtifact := openArtifactStore
	t.Cleanup(func() {
		openJournalStore = originalOpenJournal
		openProjectionStore = originalOpenProjection
		openArtifactStore = originalOpenArtifact
	})

	openJournalStore = func(path string, catalog *event.Catalog) (journalStore, error) {
		return journal, nil
	}
	openProjectionStore = func(path string) (projectionStore, error) {
		return projections, nil
	}
	openArtifactStore = func(path string) (artifactStore, error) {
		return artifacts, nil
	}

	bundle, err := Open(manifest, Paths{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	err = bundle.Close()
	if !errors.Is(err, errJournal) || !errors.Is(err, errProjection) || !errors.Is(err, errArtifact) {
		t.Fatalf("Close() error = %v, want joined close errors", err)
	}
	if !journal.closed || !projections.closed || !artifacts.closed {
		t.Fatalf("Close() should close every child store, got journal=%t projections=%t artifacts=%t", journal.closed, projections.closed, artifacts.closed)
	}
}

type fakeJournalStore struct {
	closed   bool
	closeErr error
}

func (s *fakeJournalStore) AppendCommits(_ context.Context, _ string, _ []service.PreparedCommit, _ func() time.Time) ([]event.Record, error) {
	return nil, nil
}

func (s *fakeJournalStore) List(_ context.Context, _ string) ([]event.Record, bool, error) {
	return nil, false, nil
}

func (s *fakeJournalStore) ListAfter(_ context.Context, _ string, _ uint64) ([]event.Record, bool, error) {
	return nil, false, nil
}

func (s *fakeJournalStore) HeadSeq(_ context.Context, _ string) (uint64, bool, error) {
	return 0, false, nil
}

func (s *fakeJournalStore) SubscribeAfter(_ context.Context, _ string, _ uint64) (service.EventSubscription, error) {
	return service.EventSubscription{Close: func() {}}, nil
}

func (s *fakeJournalStore) Close() error {
	s.closed = true
	return s.closeErr
}

type fakeProjectionStore struct {
	closed   bool
	closeErr error
}

func (s *fakeProjectionStore) GetProjection(_ context.Context, _ string) (service.ProjectionSnapshot, bool, error) {
	return service.ProjectionSnapshot{}, false, nil
}

func (s *fakeProjectionStore) SaveProjection(_ context.Context, _ service.ProjectionSnapshot) error {
	return nil
}

func (s *fakeProjectionStore) GetWatermark(_ context.Context, _ string) (service.ProjectionWatermark, bool, error) {
	return service.ProjectionWatermark{}, false, nil
}

func (s *fakeProjectionStore) SaveWatermark(_ context.Context, _ service.ProjectionWatermark) error {
	return nil
}

func (s *fakeProjectionStore) SaveProjectionAndWatermark(ctx context.Context, snapshot service.ProjectionSnapshot, watermark service.ProjectionWatermark) error {
	if err := s.SaveProjection(ctx, snapshot); err != nil {
		return err
	}
	return s.SaveWatermark(ctx, watermark)
}

func (s *fakeProjectionStore) ListCampaignsBySubject(_ context.Context, _ string, _ int) ([]service.CampaignSummary, error) {
	return nil, nil
}

func (s *fakeProjectionStore) Close() error {
	s.closed = true
	return s.closeErr
}

type fakeArtifactStore struct {
	closed   bool
	closeErr error
}

func (s *fakeArtifactStore) PutArtifact(_ context.Context, _ service.Artifact) error {
	return nil
}

func (s *fakeArtifactStore) GetArtifact(_ context.Context, _ string, _ string) (service.Artifact, bool, error) {
	return service.Artifact{}, false, nil
}

func (s *fakeArtifactStore) ListArtifacts(_ context.Context, _ string) ([]service.Artifact, error) {
	return nil, nil
}

func (s *fakeArtifactStore) Close() error {
	s.closed = true
	return s.closeErr
}
