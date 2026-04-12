package projections

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
	"github.com/fracturing-space/game/internal/storage/sqlite/internalutil"
	"github.com/fracturing-space/game/internal/storage/sqlite/migrations"
)

var fixedRecordTime = time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)

func TestStorePersistsProjectionAndWatermark(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projections.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}

	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "Autumn Twilight"
	state.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
		Active: true,
	}

	if err := store.SaveProjectionAndWatermark(context.Background(), service.ProjectionSnapshot{
		CampaignID:     "camp-1",
		HeadSeq:        7,
		State:          state,
		UpdatedAt:      fixedRecordTime,
		LastActivityAt: fixedRecordTime,
	}, service.ProjectionWatermark{
		CampaignID:      "camp-1",
		AppliedSeq:      7,
		ExpectedNextSeq: 8,
		UpdatedAt:       fixedRecordTime.Add(time.Minute),
	}); err != nil {
		t.Fatalf("SaveProjectionAndWatermark() error = %v", err)
	}

	state.Name = "mutated after save"
	snapshot, ok, err := store.GetProjection(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetProjection() error = %v", err)
	}
	if !ok {
		t.Fatal("GetProjection() = missing, want stored snapshot")
	}
	if got, want := snapshot.State.Name, "Autumn Twilight"; got != want {
		t.Fatalf("snapshot state name = %q, want %q", got, want)
	}
	if got, want := snapshot.LastActivityAt, fixedRecordTime; !got.Equal(want) {
		t.Fatalf("snapshot last activity at = %v, want %v", got, want)
	}

	snapshot.State.Name = "mutated after load"
	again, ok, err := store.GetProjection(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetProjection(second) error = %v", err)
	}
	if !ok || again.State.Name != "Autumn Twilight" {
		t.Fatalf("GetProjection(second) = (%q,%t), want stored clone", again.State.Name, ok)
	}

	watermark, ok, err := store.GetWatermark(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetWatermark() error = %v", err)
	}
	if !ok || watermark.ExpectedNextSeq != 8 {
		t.Fatalf("GetWatermark() = (%+v,%t), want expected next seq 8", watermark, ok)
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

	reloaded, ok, err := store.GetProjection(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetProjection(reloaded) error = %v", err)
	}
	if !ok || reloaded.HeadSeq != 7 {
		t.Fatalf("GetProjection(reloaded) = (%+v,%t), want head seq 7", reloaded, ok)
	}

	items, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject() error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("ListCampaignsBySubject() len = %d, want %d", got, want)
	}
	if got, want := items[0].CampaignID, "camp-1"; got != want {
		t.Fatalf("ListCampaignsBySubject()[0].CampaignID = %q, want %q", got, want)
	}
}

func TestStoreListsCampaignsBySubjectOrderedAndBackfillsLegacySnapshots(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projections.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}

	readyState := campaign.NewState()
	readyState.Exists = true
	readyState.CampaignID = "camp-2"
	readyState.Name = "Ready"
	readyState.AIAgentID = "agent-1"
	readyState.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
	readyState.ActiveSessionID = "sess-1"
	readyState.ActiveSceneID = "scene-1"
	readyState.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
		Active: true,
	}
	readyState.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening Scene", Active: true}
	readyState.Characters["char-1"] = character.Record{
		ID:            "char-1",
		ParticipantID: "owner-1",
		Name:          "Luna", Active: true,
	}
	if err := store.SaveProjection(context.Background(), service.ProjectionSnapshot{
		CampaignID:     "camp-2",
		HeadSeq:        2,
		State:          readyState,
		UpdatedAt:      fixedRecordTime.Add(time.Minute),
		LastActivityAt: fixedRecordTime.Add(time.Minute),
	}); err != nil {
		t.Fatalf("SaveProjection(camp-2) error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	sqlDB, err := internalutil.Open(path)
	if err != nil {
		t.Fatalf("internalutil.Open() error = %v", err)
	}
	if err := internalutil.ApplyMigrations(sqlDB, migrations.ProjectionsFS, "projections", time.Now); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}
	legacyState := campaign.NewState()
	legacyState.Exists = true
	legacyState.CampaignID = "camp-1"
	legacyState.Name = "Legacy"
	legacyState.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
		Active: true,
	}
	stateBlob, err := encodeCampaignState(legacyState)
	if err != nil {
		t.Fatalf("encodeCampaignState() error = %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO projection_snapshots (campaign_id, head_seq, state_blob, updated_at_ns)
		VALUES (?, ?, ?, ?)
	`, "camp-1", 1, stateBlob, fixedRecordTime.UnixNano()); err != nil {
		t.Fatalf("insert legacy snapshot error = %v", err)
	}
	if _, err := sqlDB.Exec(`DELETE FROM projection_campaign_summaries WHERE campaign_id = ?`, "camp-1"); err != nil {
		t.Fatalf("delete legacy summary error = %v", err)
	}
	if _, err := sqlDB.Exec(`DELETE FROM projection_campaign_subjects WHERE campaign_id = ?`, "camp-1"); err != nil {
		t.Fatalf("delete legacy subjects error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close() error = %v", err)
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

	items, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject() error = %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("ListCampaignsBySubject() len = %d, want %d", got, want)
	}
	if got, want := items[0].CampaignID, "camp-2"; got != want {
		t.Fatalf("ListCampaignsBySubject()[0].CampaignID = %q, want %q", got, want)
	}
	if !items[0].ReadyToPlay {
		t.Fatal("ListCampaignsBySubject()[0].ReadyToPlay = false, want true")
	}
	if got, want := items[1].CampaignID, "camp-1"; got != want {
		t.Fatalf("ListCampaignsBySubject()[1].CampaignID = %q, want %q", got, want)
	}
}

func TestStoreOpenBackfillPropagatesDecodeFailures(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projections.db")
	sqlDB, err := internalutil.Open(path)
	if err != nil {
		t.Fatalf("internalutil.Open() error = %v", err)
	}
	if err := internalutil.ApplyMigrations(sqlDB, migrations.ProjectionsFS, "projections", time.Now); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO projection_snapshots (campaign_id, head_seq, state_blob, updated_at_ns)
		VALUES (?, ?, ?, ?)
	`, "camp-1", 1, []byte("broken"), fixedRecordTime.UnixNano()); err != nil {
		t.Fatalf("insert broken snapshot error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}

	if _, err := Open(path); err == nil {
		t.Fatal("Open(broken legacy snapshot) error = nil, want failure")
	}
}

func TestOpenPropagatesSQLiteSeamFailures(t *testing.T) {
	t.Run("open failure", func(t *testing.T) {
		originalOpenSQLiteDB := openSQLiteDB
		originalApplyMigrations := applyMigrations
		originalBackfillCampaignLists := backfillCampaignLists
		t.Cleanup(func() {
			openSQLiteDB = originalOpenSQLiteDB
			applyMigrations = originalApplyMigrations
			backfillCampaignLists = originalBackfillCampaignLists
		})

		openErr := errors.New("sqlite open failed")
		openSQLiteDB = func(path string) (*sql.DB, error) {
			return nil, openErr
		}

		if _, err := Open(filepath.Join(t.TempDir(), "projections.db")); err == nil || !errors.Is(err, openErr) {
			t.Fatalf("Open(open failure) error = %v, want wrapped sqlite open failure", err)
		}
	})

	t.Run("migration failure", func(t *testing.T) {
		originalOpenSQLiteDB := openSQLiteDB
		originalApplyMigrations := applyMigrations
		originalBackfillCampaignLists := backfillCampaignLists
		t.Cleanup(func() {
			openSQLiteDB = originalOpenSQLiteDB
			applyMigrations = originalApplyMigrations
			backfillCampaignLists = originalBackfillCampaignLists
		})

		migrationErr := errors.New("migration failed")
		applyMigrations = func(sqlDB *sql.DB) error {
			return migrationErr
		}

		if _, err := Open(filepath.Join(t.TempDir(), "projections.db")); err == nil || !errors.Is(err, migrationErr) {
			t.Fatalf("Open(migration failure) error = %v, want wrapped migration failure", err)
		}
	})

	t.Run("backfill failure", func(t *testing.T) {
		originalOpenSQLiteDB := openSQLiteDB
		originalApplyMigrations := applyMigrations
		originalBackfillCampaignLists := backfillCampaignLists
		t.Cleanup(func() {
			openSQLiteDB = originalOpenSQLiteDB
			applyMigrations = originalApplyMigrations
			backfillCampaignLists = originalBackfillCampaignLists
		})

		backfillErr := errors.New("backfill failed")
		backfillCampaignLists = func(store *Store) error {
			return backfillErr
		}

		if _, err := Open(filepath.Join(t.TempDir(), "projections.db")); err == nil || !errors.Is(err, backfillErr) {
			t.Fatalf("Open(backfill failure) error = %v, want wrapped backfill failure", err)
		}
	})
}

func TestStoreMissingRowsAndNoOpSubjectQueries(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projections.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	snapshot, ok, err := store.GetProjection(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetProjection(missing) error = %v", err)
	}
	if ok {
		t.Fatalf("GetProjection(missing) ok = %t, want false", ok)
	}
	if snapshot.CampaignID != "" || snapshot.HeadSeq != 0 || snapshot.State.Exists {
		t.Fatalf("GetProjection(missing) = %+v, want zero snapshot", snapshot)
	}

	watermark, ok, err := store.GetWatermark(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetWatermark(missing) error = %v", err)
	}
	if ok || watermark != (service.ProjectionWatermark{}) {
		t.Fatalf("GetWatermark(missing) = (%+v,%t), want zero,false", watermark, ok)
	}

	if _, err := store.ListCampaignsBySubject(context.Background(), "   ", 10); err == nil {
		t.Fatal("ListCampaignsBySubject(blank subject) error = nil, want failure")
	}

	items, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 0)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject(zero limit) error = %v", err)
	}
	if items != nil {
		t.Fatalf("ListCampaignsBySubject(zero limit) = %v, want nil", items)
	}
}

func TestStoreRejectsNonCanonicalBoundaryInputs(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projections.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if _, _, err := store.GetProjection(context.Background(), " camp-1 "); err == nil {
		t.Fatal("GetProjection(padded campaign) error = nil, want failure")
	}
	if err := store.SaveProjection(context.Background(), service.ProjectionSnapshot{CampaignID: " camp-1 "}); err == nil {
		t.Fatal("SaveProjection(padded campaign) error = nil, want failure")
	}
	if _, _, err := store.GetWatermark(context.Background(), " camp-1 "); err == nil {
		t.Fatal("GetWatermark(padded campaign) error = nil, want failure")
	}
	if err := store.SaveWatermark(context.Background(), service.ProjectionWatermark{CampaignID: " camp-1 "}); err == nil {
		t.Fatal("SaveWatermark(padded campaign) error = nil, want failure")
	}
	if err := store.SaveProjectionAndWatermark(
		context.Background(),
		service.ProjectionSnapshot{CampaignID: " camp-1 "},
		service.ProjectionWatermark{CampaignID: " camp-1 "},
	); err == nil {
		t.Fatal("SaveProjectionAndWatermark(padded campaign) error = nil, want failure")
	}
	if _, err := store.ListCampaignsBySubject(context.Background(), " subject-1 ", 10); err == nil {
		t.Fatal("ListCampaignsBySubject(padded subject) error = nil, want failure")
	}
	if _, err := store.ListCampaignsBySubject(context.Background(), "   ", 10); err == nil {
		t.Fatal("ListCampaignsBySubject(blank subject) error = nil, want failure")
	}
}
