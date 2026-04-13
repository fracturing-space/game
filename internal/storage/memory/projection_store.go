package memory

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/fracturing-space/game/internal/canonical"
	"github.com/fracturing-space/game/internal/service"
)

// NewProjectionStore returns an in-memory projection store implementation.
func NewProjectionStore() service.ProjectionStore {
	return &projectionStore{
		projections: make(map[string]service.ProjectionSnapshot),
		watermarks:  make(map[string]service.ProjectionWatermark),
		summaries:   make(map[string]service.CampaignSummary),
		subjects:    make(map[string]map[string]struct{}),
	}
}

type projectionStore struct {
	mu          sync.RWMutex
	projections map[string]service.ProjectionSnapshot
	watermarks  map[string]service.ProjectionWatermark
	summaries   map[string]service.CampaignSummary
	subjects    map[string]map[string]struct{}
}

func (s *projectionStore) GetProjection(_ context.Context, campaignID string) (service.ProjectionSnapshot, bool, error) {
	if err := canonical.ValidateExact(campaignID, "campaign id", fmt.Errorf); err != nil {
		return service.ProjectionSnapshot{}, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.projections[campaignID]
	if !ok {
		return service.ProjectionSnapshot{}, false, nil
	}
	item.State = item.State.Clone()
	return item, true, nil
}

func (s *projectionStore) SaveProjection(_ context.Context, snapshot service.ProjectionSnapshot) error {
	if err := canonical.ValidateExact(snapshot.CampaignID, "campaign id", fmt.Errorf); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot.State = snapshot.State.Clone()
	snapshot.LastActivityAt = snapshot.LastActivityAt.UTC()
	s.projections[snapshot.CampaignID] = snapshot
	s.summaries[snapshot.CampaignID] = service.CampaignSummaryFromSnapshot(snapshot)
	for subjectID, campaignIDs := range s.subjects {
		delete(campaignIDs, snapshot.CampaignID)
		if len(campaignIDs) == 0 {
			delete(s.subjects, subjectID)
		}
	}
	for _, subjectID := range service.BoundSubjectIDs(snapshot.State) {
		bySubject, ok := s.subjects[subjectID]
		if !ok {
			bySubject = make(map[string]struct{})
			s.subjects[subjectID] = bySubject
		}
		bySubject[snapshot.CampaignID] = struct{}{}
	}
	return nil
}

func (s *projectionStore) GetWatermark(_ context.Context, campaignID string) (service.ProjectionWatermark, bool, error) {
	if err := canonical.ValidateExact(campaignID, "campaign id", fmt.Errorf); err != nil {
		return service.ProjectionWatermark{}, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.watermarks[campaignID]
	return item, ok, nil
}

func (s *projectionStore) SaveWatermark(_ context.Context, watermark service.ProjectionWatermark) error {
	if err := canonical.ValidateExact(watermark.CampaignID, "campaign id", fmt.Errorf); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watermarks[watermark.CampaignID] = watermark
	return nil
}

func (s *projectionStore) SaveProjectionAndWatermark(_ context.Context, snapshot service.ProjectionSnapshot, watermark service.ProjectionWatermark) error {
	if snapshot.CampaignID != watermark.CampaignID {
		return fmt.Errorf("projection snapshot and watermark must target the same campaign")
	}
	if err := canonical.ValidateExact(snapshot.CampaignID, "campaign id", fmt.Errorf); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot.State = snapshot.State.Clone()
	snapshot.LastActivityAt = snapshot.LastActivityAt.UTC()
	s.projections[snapshot.CampaignID] = snapshot
	s.summaries[snapshot.CampaignID] = service.CampaignSummaryFromSnapshot(snapshot)
	for subjectID, campaignIDs := range s.subjects {
		delete(campaignIDs, snapshot.CampaignID)
		if len(campaignIDs) == 0 {
			delete(s.subjects, subjectID)
		}
	}
	for _, subjectID := range service.BoundSubjectIDs(snapshot.State) {
		bySubject, ok := s.subjects[subjectID]
		if !ok {
			bySubject = make(map[string]struct{})
			s.subjects[subjectID] = bySubject
		}
		bySubject[snapshot.CampaignID] = struct{}{}
	}
	s.watermarks[watermark.CampaignID] = watermark
	return nil
}

func (s *projectionStore) ListCampaignsBySubject(_ context.Context, subjectID string, limit int) ([]service.CampaignSummary, error) {
	if limit <= 0 {
		return nil, nil
	}
	if err := canonical.ValidateExact(subjectID, "subject id", fmt.Errorf); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	bySubject, ok := s.subjects[subjectID]
	if !ok {
		return nil, nil
	}
	items := make([]service.CampaignSummary, 0, len(bySubject))
	for campaignID := range bySubject {
		summary, ok := s.summaries[campaignID]
		if !ok {
			continue
		}
		items = append(items, summary)
	}
	slices.SortFunc(items, service.CompareCampaignSummary)
	if len(items) > limit {
		items = items[:limit]
	}
	return append([]service.CampaignSummary(nil), items...), nil
}
