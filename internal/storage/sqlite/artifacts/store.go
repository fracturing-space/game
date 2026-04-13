package artifacts

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

// Store implements the service artifact store port on SQLite.
type Store struct {
	db      *sql.DB
	q       *db.Queries
	writeMu sync.Mutex
}

var (
	openSQLiteDB    = internalutil.Open
	applyMigrations = func(sqlDB *sql.DB) error {
		return internalutil.ApplyMigrations(sqlDB, migrations.ArtifactsFS, "artifacts", time.Now)
	}
)

// Open opens a SQLite artifact store at the provided path.
func Open(path string) (*Store, error) {
	sqlDB, err := openSQLiteDB(path)
	if err != nil {
		return nil, fmt.Errorf("open artifacts store: %w", err)
	}
	if err := applyMigrations(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate artifacts store: %w", err)
	}
	return &Store{
		db: sqlDB,
		q:  db.New(sqlDB),
	}, nil
}

// Close closes the underlying SQLite database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// PutArtifact upserts one campaign artifact.
func (s *Store) PutArtifact(ctx context.Context, item service.Artifact) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	var err error
	item.CampaignID, err = validateCampaignID(item.CampaignID)
	if err != nil {
		return err
	}
	item.Path, err = validateArtifactPath(item.Path)
	if err != nil {
		return err
	}
	return s.q.PutArtifact(ctx, db.PutArtifactParams{
		CampaignID:  item.CampaignID,
		Path:        item.Path,
		Content:     item.Content,
		UpdatedAtNs: item.UpdatedAt.UTC().UnixNano(),
	})
}

// GetArtifact loads one campaign artifact by path.
func (s *Store) GetArtifact(ctx context.Context, campaignID string, path string) (service.Artifact, bool, error) {
	var err error
	campaignID, err = validateCampaignID(campaignID)
	if err != nil {
		return service.Artifact{}, false, err
	}
	path, err = validateArtifactPath(path)
	if err != nil {
		return service.Artifact{}, false, err
	}
	row, err := s.q.GetArtifact(ctx, db.GetArtifactParams{
		CampaignID: campaignID,
		Path:       path,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return service.Artifact{}, false, nil
	}
	if err != nil {
		return service.Artifact{}, false, err
	}
	return service.Artifact{
		CampaignID: campaignID,
		Path:       path,
		Content:    row.Content,
		UpdatedAt:  time.Unix(0, row.UpdatedAtNs).UTC(),
	}, true, nil
}

// ListArtifacts returns every artifact for one campaign.
func (s *Store) ListArtifacts(ctx context.Context, campaignID string) ([]service.Artifact, error) {
	var err error
	campaignID, err = validateCampaignID(campaignID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListArtifactsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	items := make([]service.Artifact, 0, len(rows))
	for _, row := range rows {
		items = append(items, service.Artifact{
			CampaignID: campaignID,
			Path:       row.Path,
			Content:    row.Content,
			UpdatedAt:  time.Unix(0, row.UpdatedAtNs).UTC(),
		})
	}
	return items, nil
}

func validateCampaignID(campaignID string) (string, error) {
	if err := canonical.ValidateExact(campaignID, "campaign id", fmt.Errorf); err != nil {
		return "", err
	}
	return campaignID, nil
}

func validateArtifactPath(path string) (string, error) {
	if err := canonical.ValidateRelativePath(path, "artifact path", fmt.Errorf); err != nil {
		return "", err
	}
	return path, nil
}
