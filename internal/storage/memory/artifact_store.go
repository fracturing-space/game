package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/service"
)

// NewArtifactStore returns an in-memory artifact store implementation.
func NewArtifactStore() service.ArtifactStore {
	return &artifactStore{artifacts: make(map[string]map[string]service.Artifact)}
}

type artifactStore struct {
	mu        sync.RWMutex
	artifacts map[string]map[string]service.Artifact
}

func (s *artifactStore) PutArtifact(_ context.Context, item service.Artifact) error {
	var err error
	item.CampaignID, err = validateCampaignID(item.CampaignID)
	if err != nil {
		return err
	}
	item.Path, err = validateArtifactPath(item.Path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	byCampaign, ok := s.artifacts[item.CampaignID]
	if !ok {
		byCampaign = make(map[string]service.Artifact)
		s.artifacts[item.CampaignID] = byCampaign
	}
	byCampaign[item.Path] = item
	return nil
}

func (s *artifactStore) GetArtifact(_ context.Context, campaignID string, path string) (service.Artifact, bool, error) {
	var err error
	campaignID, err = validateCampaignID(campaignID)
	if err != nil {
		return service.Artifact{}, false, err
	}
	path, err = validateArtifactPath(path)
	if err != nil {
		return service.Artifact{}, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	byCampaign, ok := s.artifacts[campaignID]
	if !ok {
		return service.Artifact{}, false, nil
	}
	item, ok := byCampaign[path]
	return item, ok, nil
}

func (s *artifactStore) ListArtifacts(_ context.Context, campaignID string) ([]service.Artifact, error) {
	var err error
	campaignID, err = validateCampaignID(campaignID)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	byCampaign, ok := s.artifacts[campaignID]
	if !ok {
		return nil, nil
	}
	items := make([]service.Artifact, 0, len(byCampaign))
	for _, item := range byCampaign {
		items = append(items, item)
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
