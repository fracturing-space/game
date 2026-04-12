package service

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

type testMemoryStore struct {
	mu          sync.Mutex
	timelines   map[string][]event.Record
	subscribers map[string]map[*testSubscriptionHandle]struct{}
}

func newTestMemoryStore() *testMemoryStore {
	return &testMemoryStore{
		timelines:   make(map[string][]event.Record),
		subscribers: make(map[string]map[*testSubscriptionHandle]struct{}),
	}
}

func (s *testMemoryStore) AppendCommits(_ context.Context, campaignID string, commits []PreparedCommit, now func() time.Time) ([]event.Record, error) {
	s.mu.Lock()
	timeline := append([]event.Record(nil), s.timelines[campaignID]...)
	nextSeq := uint64(len(timeline)) + 1
	nextCommitSeq := uint64(1)
	if len(timeline) != 0 {
		nextCommitSeq = timeline[len(timeline)-1].CommitSeq + 1
	}
	records := make([]event.Record, 0)
	for _, commit := range commits {
		recordedAt := now().UTC()
		for _, envelope := range commit.Events {
			record := event.Record{
				Seq:        nextSeq,
				CommitSeq:  nextCommitSeq,
				RecordedAt: recordedAt,
				Envelope:   envelope,
			}
			records = append(records, record)
			timeline = append(timeline, record)
			nextSeq++
		}
		nextCommitSeq++
	}
	s.timelines[campaignID] = timeline
	subscribers := s.campaignSubscribersLocked(campaignID)
	s.mu.Unlock()

	cloned := append([]event.Record(nil), records...)
	s.notifySubscribers(campaignID, subscribers, cloned)
	return cloned, nil
}

func (s *testMemoryStore) List(_ context.Context, campaignID string) ([]event.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	timeline, ok := s.timelines[campaignID]
	if !ok {
		return nil, false, nil
	}
	return append([]event.Record(nil), timeline...), true, nil
}

func (s *testMemoryStore) ListAfter(_ context.Context, campaignID string, afterSeq uint64) ([]event.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	timeline, ok := s.timelines[campaignID]
	if !ok {
		return nil, false, nil
	}
	index := 0
	for index < len(timeline) && timeline[index].Seq <= afterSeq {
		index++
	}
	return append([]event.Record(nil), timeline[index:]...), true, nil
}

func (s *testMemoryStore) HeadSeq(_ context.Context, campaignID string) (uint64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	timeline, ok := s.timelines[campaignID]
	if !ok {
		return 0, false, nil
	}
	if len(timeline) == 0 {
		return 0, true, nil
	}
	return timeline[len(timeline)-1].Seq, true, nil
}

func (s *testMemoryStore) SubscribeAfter(ctx context.Context, campaignID string, afterSeq uint64) (EventSubscription, error) {
	s.mu.Lock()
	index := 0
	if timeline, ok := s.timelines[campaignID]; ok {
		for index < len(timeline) && timeline[index].Seq <= afterSeq {
			index++
		}
	}
	handle := newTestSubscriptionHandle(ctx, afterSeq, append([]event.Record(nil), s.timelines[campaignID][index:]...))
	handle.closeFn = func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if byCampaign, ok := s.subscribers[campaignID]; ok {
			delete(byCampaign, handle)
			if len(byCampaign) == 0 {
				delete(s.subscribers, campaignID)
			}
		}
	}
	if _, ok := s.subscribers[campaignID]; !ok {
		s.subscribers[campaignID] = make(map[*testSubscriptionHandle]struct{})
	}
	s.subscribers[campaignID][handle] = struct{}{}
	s.mu.Unlock()
	return handle.subscription(), nil
}

func (s *testMemoryStore) notifySubscribers(campaignID string, subscribers []*testSubscriptionHandle, records []event.Record) {
	for _, sub := range subscribers {
		for _, record := range records {
			if !sub.enqueue(record) {
				s.closeSubscriber(campaignID, sub)
				goto nextSubscriber
			}
		}
	nextSubscriber:
	}
}

func (s *testMemoryStore) campaignSubscribersLocked(campaignID string) []*testSubscriptionHandle {
	byCampaign, ok := s.subscribers[campaignID]
	if !ok {
		return nil
	}
	out := make([]*testSubscriptionHandle, 0, len(byCampaign))
	for handle := range byCampaign {
		out = append(out, handle)
	}
	return out
}

func (s *testMemoryStore) closeSubscriber(campaignID string, handle *testSubscriptionHandle) {
	if handle == nil {
		return
	}
	s.mu.Lock()
	if byCampaign, ok := s.subscribers[campaignID]; ok {
		delete(byCampaign, handle)
		if len(byCampaign) == 0 {
			delete(s.subscribers, campaignID)
		}
	}
	s.mu.Unlock()
	handle.closeRecords()
}

type testProjectionStore struct {
	projections map[string]ProjectionSnapshot
	watermarks  map[string]ProjectionWatermark
}

func newTestProjectionStore() *testProjectionStore {
	return &testProjectionStore{
		projections: make(map[string]ProjectionSnapshot),
		watermarks:  make(map[string]ProjectionWatermark),
	}
}

func (s *testProjectionStore) GetProjection(_ context.Context, campaignID string) (ProjectionSnapshot, bool, error) {
	item, ok := s.projections[strings.TrimSpace(campaignID)]
	if !ok {
		return ProjectionSnapshot{}, false, nil
	}
	item.State = item.State.Clone()
	return item, true, nil
}

func (s *testProjectionStore) SaveProjection(_ context.Context, snapshot ProjectionSnapshot) error {
	snapshot.CampaignID = strings.TrimSpace(snapshot.CampaignID)
	if snapshot.CampaignID == "" {
		return nil
	}
	snapshot.State = snapshot.State.Clone()
	snapshot.LastActivityAt = snapshot.LastActivityAt.UTC()
	s.projections[snapshot.CampaignID] = snapshot
	return nil
}

func (s *testProjectionStore) GetWatermark(_ context.Context, campaignID string) (ProjectionWatermark, bool, error) {
	item, ok := s.watermarks[strings.TrimSpace(campaignID)]
	return item, ok, nil
}

func (s *testProjectionStore) SaveWatermark(_ context.Context, watermark ProjectionWatermark) error {
	watermark.CampaignID = strings.TrimSpace(watermark.CampaignID)
	if watermark.CampaignID == "" {
		return nil
	}
	s.watermarks[watermark.CampaignID] = watermark
	return nil
}

func (s *testProjectionStore) SaveProjectionAndWatermark(ctx context.Context, snapshot ProjectionSnapshot, watermark ProjectionWatermark) error {
	if err := s.SaveProjection(ctx, snapshot); err != nil {
		return err
	}
	return s.SaveWatermark(ctx, watermark)
}

func (s *testProjectionStore) ListCampaignsBySubject(_ context.Context, subjectID string, limit int) ([]CampaignSummary, error) {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" || limit <= 0 {
		return nil, nil
	}
	summaries := make([]CampaignSummary, 0, len(s.projections))
	for _, snapshot := range s.projections {
		for _, candidate := range BoundSubjectIDs(snapshot.State) {
			if candidate != subjectID {
				continue
			}
			summaries = append(summaries, CampaignSummaryFromSnapshot(snapshot))
			break
		}
	}
	slices.SortFunc(summaries, CompareCampaignSummary)
	if len(summaries) > limit {
		summaries = summaries[:limit]
	}
	return append([]CampaignSummary(nil), summaries...), nil
}

type testArtifactStore struct {
	artifacts map[string]map[string]Artifact
}

func newTestArtifactStore() *testArtifactStore {
	return &testArtifactStore{artifacts: make(map[string]map[string]Artifact)}
}

func (s *testArtifactStore) PutArtifact(_ context.Context, item Artifact) error {
	item.CampaignID = strings.TrimSpace(item.CampaignID)
	item.Path = normalizeTestArtifactPath(item.Path)
	if item.CampaignID == "" || item.Path == "" {
		return nil
	}
	byCampaign, ok := s.artifacts[item.CampaignID]
	if !ok {
		byCampaign = make(map[string]Artifact)
		s.artifacts[item.CampaignID] = byCampaign
	}
	byCampaign[item.Path] = item
	return nil
}

func (s *testArtifactStore) GetArtifact(_ context.Context, campaignID string, path string) (Artifact, bool, error) {
	byCampaign, ok := s.artifacts[strings.TrimSpace(campaignID)]
	if !ok {
		return Artifact{}, false, nil
	}
	item, ok := byCampaign[normalizeTestArtifactPath(path)]
	return item, ok, nil
}

func (s *testArtifactStore) ListArtifacts(_ context.Context, campaignID string) ([]Artifact, error) {
	byCampaign, ok := s.artifacts[strings.TrimSpace(campaignID)]
	if !ok {
		return nil, nil
	}
	items := make([]Artifact, 0, len(byCampaign))
	for _, item := range byCampaign {
		items = append(items, item)
	}
	return items, nil
}

func normalizeTestArtifactPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	return path
}

type testSubscriptionHandle struct {
	records     chan event.Record
	live        chan event.Record
	ctx         context.Context
	stop        chan struct{}
	once        sync.Once
	recordsOnce sync.Once
	closeFn     func()
}

func newTestSubscriptionHandle(ctx context.Context, afterSeq uint64, initial []event.Record) *testSubscriptionHandle {
	if ctx == nil {
		ctx = context.Background()
	}
	handle := &testSubscriptionHandle{
		records: make(chan event.Record, 32),
		live:    make(chan event.Record, 32),
		ctx:     ctx,
		stop:    make(chan struct{}),
	}
	go handle.run(afterSeq, initial)
	return handle
}

func (h *testSubscriptionHandle) subscription() EventSubscription {
	if h == nil {
		return EventSubscription{
			Records: nil,
			Close:   func() {},
		}
	}
	return EventSubscription{
		Records: h.records,
		Close: func() {
			h.once.Do(func() {
				if h.closeFn != nil {
					h.closeFn()
				}
				h.closeRecords()
			})
		},
	}
}

func (h *testSubscriptionHandle) enqueue(record event.Record) bool {
	if h == nil {
		return false
	}
	select {
	case <-h.stop:
		return false
	default:
	}
	select {
	case h.live <- record:
		return true
	default:
		return false
	}
}

func (h *testSubscriptionHandle) run(afterSeq uint64, initial []event.Record) {
	if h == nil || h.records == nil {
		return
	}
	defer close(h.records)

	lastSeq := afterSeq
	for _, record := range initial {
		if record.Seq <= lastSeq {
			continue
		}
		record = cloneTestRecord(record)
		select {
		case <-h.ctx.Done():
			return
		case <-h.stop:
			return
		case h.records <- record:
			lastSeq = record.Seq
		}
	}
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-h.stop:
			return
		case record, ok := <-h.live:
			if !ok {
				return
			}
			if record.Seq <= lastSeq {
				continue
			}
			record = cloneTestRecord(record)
			select {
			case <-h.ctx.Done():
				return
			case <-h.stop:
				return
			case h.records <- record:
				lastSeq = record.Seq
			}
		}
	}
}

func (h *testSubscriptionHandle) closeRecords() {
	if h == nil || h.stop == nil {
		return
	}
	h.recordsOnce.Do(func() {
		close(h.stop)
	})
}

func cloneTestRecord(record event.Record) event.Record {
	return event.Record{
		Seq:        record.Seq,
		CommitSeq:  record.CommitSeq,
		RecordedAt: record.RecordedAt,
		Envelope: event.Envelope{
			CampaignID: record.CampaignID,
			Message:    cloneTestMessage(record.Message),
		},
	}
}

func cloneTestMessage(message event.Message) event.Message {
	switch typed := message.(type) {
	case scene.Created:
		typed.CharacterIDs = append([]string(nil), typed.CharacterIDs...)
		return typed
	case scene.CastReplaced:
		typed.CharacterIDs = append([]string(nil), typed.CharacterIDs...)
		return typed
	case session.Started:
		typed.CharacterControllers = session.CloneAssignments(typed.CharacterControllers)
		return typed
	case session.Ended:
		typed.CharacterControllers = session.CloneAssignments(typed.CharacterControllers)
		return typed
	default:
		return message
	}
}
