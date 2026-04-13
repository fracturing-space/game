package projections

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/storage/sqlite/db"
	"github.com/fracturing-space/game/internal/storage/sqlite/internalutil"
	"github.com/fracturing-space/game/internal/storage/sqlite/migrations"
)

// Store implements the service projection store port on SQLite.
type Store struct {
	db      *sql.DB
	q       *db.Queries
	writeMu sync.Mutex
}

var (
	openSQLiteDB    = internalutil.Open
	applyMigrations = func(sqlDB *sql.DB) error {
		return internalutil.ApplyMigrations(sqlDB, migrations.ProjectionsFS, "projections", time.Now)
	}
	backfillCampaignLists = func(store *Store) error {
		return store.backfillLegacyCampaignLists()
	}
)

// Open opens a SQLite projection store at the provided path.
func Open(path string) (*Store, error) {
	sqlDB, err := openSQLiteDB(path)
	if err != nil {
		return nil, fmt.Errorf("open projections store: %w", err)
	}
	if err := applyMigrations(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate projections store: %w", err)
	}
	store := &Store{
		db: sqlDB,
		q:  db.New(sqlDB),
	}
	if err := backfillCampaignLists(store); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("backfill projection campaign lists: %w", err)
	}
	return store, nil
}

// Close closes the underlying SQLite database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// GetProjection loads one campaign snapshot.
func (s *Store) GetProjection(ctx context.Context, campaignID string) (service.ProjectionSnapshot, bool, error) {
	if err := canonical.ValidateExact(campaignID, "campaign id", fmt.Errorf); err != nil {
		return service.ProjectionSnapshot{}, false, err
	}
	row, err := s.q.GetProjection(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ProjectionSnapshot{}, false, nil
	}
	if err != nil {
		return service.ProjectionSnapshot{}, false, fmt.Errorf("get projection %s: %w", campaignID, err)
	}
	state, err := decodeCampaignState(row.StateBlob)
	if err != nil {
		return service.ProjectionSnapshot{}, false, fmt.Errorf("decode projection %s: %w", campaignID, err)
	}
	return service.ProjectionSnapshot{
		CampaignID:     campaignID,
		HeadSeq:        uint64(row.HeadSeq),
		State:          state,
		UpdatedAt:      time.Unix(0, row.UpdatedAtNs).UTC(),
		LastActivityAt: time.Unix(0, row.LastActivityAtNs).UTC(),
	}, true, nil
}

// SaveProjection persists one campaign snapshot.
func (s *Store) SaveProjection(ctx context.Context, snapshot service.ProjectionSnapshot) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := canonical.ValidateExact(snapshot.CampaignID, "campaign id", fmt.Errorf); err != nil {
		return err
	}

	stateBlob, err := encodeCampaignState(snapshot.State.Clone())
	if err != nil {
		return fmt.Errorf("encode projection %s: %w", snapshot.CampaignID, err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin projection transaction %s: %w", snapshot.CampaignID, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := s.q.WithTx(tx)
	if err = qtx.PutProjection(ctx, db.PutProjectionParams{
		CampaignID:       snapshot.CampaignID,
		HeadSeq:          int64(snapshot.HeadSeq),
		StateBlob:        stateBlob,
		UpdatedAtNs:      snapshot.UpdatedAt.UTC().UnixNano(),
		LastActivityAtNs: snapshot.LastActivityAt.UTC().UnixNano(),
	}); err != nil {
		return fmt.Errorf("save projection snapshot %s: %w", snapshot.CampaignID, err)
	}
	if err = putCampaignListProjection(ctx, qtx, snapshot); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit projection transaction %s: %w", snapshot.CampaignID, err)
	}
	return nil
}

// GetWatermark loads one projection watermark.
func (s *Store) GetWatermark(ctx context.Context, campaignID string) (service.ProjectionWatermark, bool, error) {
	if err := canonical.ValidateExact(campaignID, "campaign id", fmt.Errorf); err != nil {
		return service.ProjectionWatermark{}, false, err
	}
	row, err := s.q.GetWatermark(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ProjectionWatermark{}, false, nil
	}
	if err != nil {
		return service.ProjectionWatermark{}, false, fmt.Errorf("get projection watermark %s: %w", campaignID, err)
	}
	return service.ProjectionWatermark{
		CampaignID:      campaignID,
		AppliedSeq:      uint64(row.AppliedSeq),
		ExpectedNextSeq: uint64(row.ExpectedNextSeq),
		UpdatedAt:       time.Unix(0, row.UpdatedAtNs).UTC(),
	}, true, nil
}

// SaveWatermark persists one projection watermark.
func (s *Store) SaveWatermark(ctx context.Context, watermark service.ProjectionWatermark) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := canonical.ValidateExact(watermark.CampaignID, "campaign id", fmt.Errorf); err != nil {
		return err
	}

	if err := s.q.PutWatermark(ctx, db.PutWatermarkParams{
		CampaignID:      watermark.CampaignID,
		AppliedSeq:      int64(watermark.AppliedSeq),
		ExpectedNextSeq: int64(watermark.ExpectedNextSeq),
		UpdatedAtNs:     watermark.UpdatedAt.UTC().UnixNano(),
	}); err != nil {
		return fmt.Errorf("save projection watermark %s: %w", watermark.CampaignID, err)
	}
	return nil
}

// SaveProjectionAndWatermark persists one projection snapshot and watermark atomically.
func (s *Store) SaveProjectionAndWatermark(ctx context.Context, snapshot service.ProjectionSnapshot, watermark service.ProjectionWatermark) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if snapshot.CampaignID != watermark.CampaignID {
		return fmt.Errorf("projection snapshot and watermark must target the same campaign")
	}
	if err := canonical.ValidateExact(snapshot.CampaignID, "campaign id", fmt.Errorf); err != nil {
		return err
	}

	stateBlob, err := encodeCampaignState(snapshot.State.Clone())
	if err != nil {
		return fmt.Errorf("encode projection %s: %w", snapshot.CampaignID, err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin projection transaction %s: %w", snapshot.CampaignID, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := s.q.WithTx(tx)
	if err = qtx.PutProjection(ctx, db.PutProjectionParams{
		CampaignID:       snapshot.CampaignID,
		HeadSeq:          int64(snapshot.HeadSeq),
		StateBlob:        stateBlob,
		UpdatedAtNs:      snapshot.UpdatedAt.UTC().UnixNano(),
		LastActivityAtNs: snapshot.LastActivityAt.UTC().UnixNano(),
	}); err != nil {
		return fmt.Errorf("save projection snapshot %s: %w", snapshot.CampaignID, err)
	}
	if err = putCampaignListProjection(ctx, qtx, snapshot); err != nil {
		return err
	}
	if err = qtx.PutWatermark(ctx, db.PutWatermarkParams{
		CampaignID:      watermark.CampaignID,
		AppliedSeq:      int64(watermark.AppliedSeq),
		ExpectedNextSeq: int64(watermark.ExpectedNextSeq),
		UpdatedAtNs:     watermark.UpdatedAt.UTC().UnixNano(),
	}); err != nil {
		return fmt.Errorf("save projection watermark %s: %w", watermark.CampaignID, err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit projection transaction %s: %w", snapshot.CampaignID, err)
	}
	return nil
}

// ListCampaignsBySubject returns launcher summaries for campaigns visible to the subject.
func (s *Store) ListCampaignsBySubject(ctx context.Context, subjectID string, limit int) ([]service.CampaignSummary, error) {
	if limit <= 0 {
		return nil, nil
	}
	if err := canonical.ValidateExact(subjectID, "subject id", fmt.Errorf); err != nil {
		return nil, err
	}
	rows, err := s.q.ListCampaignsBySubject(ctx, db.ListCampaignsBySubjectParams{
		SubjectID: subjectID,
		Limit:     int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list campaign summaries for subject %s: %w", subjectID, err)
	}
	items := make([]service.CampaignSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, service.CampaignSummary{
			CampaignID:       row.CampaignID,
			Name:             row.Name,
			ReadyToPlay:      row.ReadyToPlay != 0,
			HasAIBinding:     row.HasAiBinding != 0,
			HasActiveSession: row.HasActiveSession != 0,
			LastActivityAt:   time.Unix(0, row.LastActivityAtNs).UTC(),
		})
	}
	return items, nil
}

func (s *Store) backfillLegacyCampaignLists() error {
	rows, err := s.q.ListProjectionSnapshotsForBackfill(context.Background())
	if err != nil {
		return fmt.Errorf("list projection snapshots for backfill: %w", err)
	}
	for _, row := range rows {
		state, err := decodeCampaignState(row.StateBlob)
		if err != nil {
			return fmt.Errorf("decode legacy projection %s: %w", row.CampaignID, err)
		}
		snapshot := service.ProjectionSnapshot{
			CampaignID:     row.CampaignID,
			HeadSeq:        uint64(row.HeadSeq),
			State:          state,
			UpdatedAt:      time.Unix(0, row.UpdatedAtNs).UTC(),
			LastActivityAt: time.Unix(0, row.LastActivityAtNs).UTC(),
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin backfill transaction %s: %w", row.CampaignID, err)
		}
		qtx := s.q.WithTx(tx)
		if err := putCampaignListProjection(context.Background(), qtx, snapshot); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit backfill transaction %s: %w", row.CampaignID, err)
		}
	}
	return nil
}

func putCampaignListProjection(ctx context.Context, qtx *db.Queries, snapshot service.ProjectionSnapshot) error {
	summary := service.CampaignSummaryFromSnapshot(snapshot)
	if err := qtx.PutCampaignSummary(ctx, db.PutCampaignSummaryParams{
		CampaignID:       summary.CampaignID,
		Name:             summary.Name,
		ReadyToPlay:      boolToInt64(summary.ReadyToPlay),
		HasAiBinding:     boolToInt64(summary.HasAIBinding),
		HasActiveSession: boolToInt64(summary.HasActiveSession),
		LastActivityAtNs: summary.LastActivityAt.UTC().UnixNano(),
		UpdatedAtNs:      snapshot.UpdatedAt.UTC().UnixNano(),
	}); err != nil {
		return fmt.Errorf("save campaign summary %s: %w", snapshot.CampaignID, err)
	}
	if err := qtx.DeleteCampaignSubjects(ctx, snapshot.CampaignID); err != nil {
		return fmt.Errorf("delete campaign subjects %s: %w", snapshot.CampaignID, err)
	}
	for _, subjectID := range service.BoundSubjectIDs(snapshot.State) {
		if err := qtx.PutCampaignSubject(ctx, db.PutCampaignSubjectParams{
			SubjectID:  subjectID,
			CampaignID: snapshot.CampaignID,
		}); err != nil {
			return fmt.Errorf("save campaign subject %s/%s: %w", subjectID, snapshot.CampaignID, err)
		}
	}
	return nil
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
