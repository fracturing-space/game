package eventjournal

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/service"
)

var fixedRecordTime = time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)

func TestStoreAppendsListsAndPersists(t *testing.T) {
	t.Parallel()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "events.db")

	store, err := Open(path, manifest.Events)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}

	subscription, err := store.SubscribeAfter(context.Background(), "camp-1", 0)
	if err != nil {
		t.Fatalf("SubscribeAfter() error = %v", err)
	}
	defer subscription.Close()

	appended, err := store.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{
		{Events: []event.Envelope{
			mustEnvelope(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{Name: "Autumn Twilight"}),
			mustEnvelope(t, participant.JoinedEventSpec, "camp-1", participant.Joined{
				ParticipantID: "part-1",
				Name:          "louis", Access: participant.AccessOwner, SubjectID: "subject-1",
			}),
		}},
		{Events: []event.Envelope{
			mustEnvelope(t, campaign.AIBoundEventSpec, "camp-1", campaign.AIBound{AIAgentID: "agent-7"}),
		}},
	}, func() time.Time {
		return fixedRecordTime
	})
	if err != nil {
		t.Fatalf("AppendCommits() error = %v", err)
	}
	if got, want := len(appended), 3; got != want {
		t.Fatalf("appended len = %d, want %d", got, want)
	}
	if got, want := appended[0].CommitSeq, uint64(1); got != want {
		t.Fatalf("first commit seq = %d, want %d", got, want)
	}
	if got, want := appended[2].CommitSeq, uint64(2); got != want {
		t.Fatalf("second commit seq = %d, want %d", got, want)
	}

	for i := range 3 {
		record := <-subscription.Records
		if got, want := record.Seq, uint64(i+1); got != want {
			t.Fatalf("subscription seq = %d, want %d", got, want)
		}
	}

	listed, ok, err := store.List(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !ok || len(listed) != 3 {
		t.Fatalf("List() = (%d,%t), want (3,true)", len(listed), ok)
	}
	after, ok, err := store.ListAfter(context.Background(), "camp-1", 1)
	if err != nil {
		t.Fatalf("ListAfter() error = %v", err)
	}
	if !ok || len(after) != 2 {
		t.Fatalf("ListAfter() = (%d,%t), want (2,true)", len(after), ok)
	}
	head, ok, err := store.HeadSeq(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("HeadSeq() error = %v", err)
	}
	if !ok || head != 3 {
		t.Fatalf("HeadSeq() = (%d,%t), want (3,true)", head, ok)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	store, err = Open(path, manifest.Events)
	if err != nil {
		t.Fatalf("Open(second) error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	}()

	reloaded, ok, err := store.List(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("List(reloaded) error = %v", err)
	}
	if !ok || len(reloaded) != 3 {
		t.Fatalf("List(reloaded) = (%d,%t), want (3,true)", len(reloaded), ok)
	}
	joined, err := event.MessageAs[participant.Joined](reloaded[1].Envelope)
	if err != nil {
		t.Fatalf("MessageAs(joined) error = %v", err)
	}
	if got, want := joined.SubjectID, "subject-1"; got != want {
		t.Fatalf("joined subject id = %q, want %q", got, want)
	}
}

func TestStoreClosesSlowSubscriberOnOverflow(t *testing.T) {
	t.Parallel()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "events.db")

	store, err := Open(path, manifest.Events)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	subscription, err := store.SubscribeAfter(context.Background(), "camp-1", 0)
	if err != nil {
		t.Fatalf("SubscribeAfter() error = %v", err)
	}
	defer subscription.Close()

	// Exceed both the live and outbound channel buffers so overflow is deterministic.
	events := make([]event.Envelope, 96)
	for i := range events {
		events[i] = mustEnvelope(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{
			Name: "Autumn Twilight",
		})
	}

	if _, err := store.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{Events: events}}, func() time.Time {
		return fixedRecordTime
	}); err != nil {
		t.Fatalf("AppendCommits() error = %v", err)
	}

	count := 0
	for range subscription.Records {
		count++
	}
	if count >= len(events) {
		t.Fatalf("delivered records = %d, want slow-subscriber close before all %d records", count, len(events))
	}
}

func TestStoreHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "events.db")

	store, err := Open(path, manifest.Events)
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

	if _, err := store.AppendCommits(ctx, "camp-1", []service.PreparedCommit{{Events: []event.Envelope{
		mustEnvelope(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{Name: "Autumn Twilight"}),
	}}}, func() time.Time {
		return fixedRecordTime
	}); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("AppendCommits(canceled) error = %v, want context canceled", err)
	}
	if _, _, err := store.List(ctx, "camp-1"); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("List(canceled) error = %v, want context canceled", err)
	}
}

func TestOpenPropagatesSQLiteOpenAndMigrationFailures(t *testing.T) {
	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

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

		if _, err := Open(filepath.Join(t.TempDir(), "events.db"), manifest.Events); err == nil || !errors.Is(err, openErr) {
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

		if _, err := Open(filepath.Join(t.TempDir(), "events.db"), manifest.Events); err == nil || !errors.Is(err, migrationErr) {
			t.Fatalf("Open(migration failure) error = %v, want wrapped migration failure", err)
		}
	})
}

func TestStoreHeadSeqAndListAfterEdgeCases(t *testing.T) {
	t.Parallel()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "events.db")
	store, err := Open(path, manifest.Events)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	head, ok, err := store.HeadSeq(context.Background(), "missing")
	if err != nil {
		t.Fatalf("HeadSeq(missing) error = %v", err)
	}
	if ok || head != 0 {
		t.Fatalf("HeadSeq(missing) = (%d,%t), want (0,false)", head, ok)
	}

	if _, err := store.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{Events: []event.Envelope{
		mustEnvelope(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{Name: "Autumn Twilight"}),
		mustEnvelope(t, participant.JoinedEventSpec, "camp-1", participant.Joined{
			ParticipantID: "part-1",
			Name:          "louis", Access: participant.AccessOwner, SubjectID: "subject-1",
		}),
	}}}, func() time.Time {
		return fixedRecordTime
	}); err != nil {
		t.Fatalf("AppendCommits() error = %v", err)
	}

	after, ok, err := store.ListAfter(context.Background(), "camp-1", 2)
	if err != nil {
		t.Fatalf("ListAfter(head) error = %v", err)
	}
	if !ok || len(after) != 0 {
		t.Fatalf("ListAfter(head) = (%d,%t), want (0,true)", len(after), ok)
	}
}

func TestSubscribeAfterReturnsErrorWhenStoreIsClosed(t *testing.T) {
	t.Parallel()

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "events.db")
	store, err := Open(path, manifest.Events)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := store.SubscribeAfter(context.Background(), "camp-1", 0); err == nil {
		t.Fatal("SubscribeAfter(closed store) error = nil, want failure")
	}
}

func mustEnvelope[T event.Message](t *testing.T, spec event.TypedSpec[T], campaignID string, payload T) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(spec, campaignID, payload)
	if err != nil {
		t.Fatalf("NewEnvelope(%s) error = %v", spec.Definition().Type, err)
	}
	return envelope
}
