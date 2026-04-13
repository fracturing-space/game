package memory

import (
	"context"
	"sync"
	"time"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/service"
)

// NewJournal returns an in-memory journal implementation.
func NewJournal() service.Journal {
	return &journal{
		timelines:   make(map[string][]event.Record),
		subscribers: make(map[string]map[*subscriptionHandle]struct{}),
	}
}

type journal struct {
	mu          sync.Mutex
	timelines   map[string][]event.Record
	subscribers map[string]map[*subscriptionHandle]struct{}
}

func (s *journal) AppendCommits(_ context.Context, campaignID string, commits []service.PreparedCommit, now func() time.Time) ([]event.Record, error) {
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

func (s *journal) List(_ context.Context, campaignID string) ([]event.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	timeline, ok := s.timelines[campaignID]
	if !ok {
		return nil, false, nil
	}
	return append([]event.Record(nil), timeline...), true, nil
}

func (s *journal) ListAfter(_ context.Context, campaignID string, afterSeq uint64) ([]event.Record, bool, error) {
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

func (s *journal) HeadSeq(_ context.Context, campaignID string) (uint64, bool, error) {
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

func (s *journal) SubscribeAfter(ctx context.Context, campaignID string, afterSeq uint64) (service.EventSubscription, error) {
	s.mu.Lock()
	index := 0
	if timeline, ok := s.timelines[campaignID]; ok {
		for index < len(timeline) && timeline[index].Seq <= afterSeq {
			index++
		}
	}
	handle := newSubscriptionHandle(ctx, afterSeq, append([]event.Record(nil), s.timelines[campaignID][index:]...))
	handle.SetCloseFunc(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if byCampaign, ok := s.subscribers[campaignID]; ok {
			delete(byCampaign, handle)
			if len(byCampaign) == 0 {
				delete(s.subscribers, campaignID)
			}
		}
	})
	if _, ok := s.subscribers[campaignID]; !ok {
		s.subscribers[campaignID] = make(map[*subscriptionHandle]struct{})
	}
	s.subscribers[campaignID][handle] = struct{}{}
	s.mu.Unlock()
	return handle.Subscription(), nil
}

func (s *journal) notifySubscribers(campaignID string, subscribers []*subscriptionHandle, records []event.Record) {
	for _, sub := range subscribers {
		for _, record := range records {
			if !sub.Enqueue(record) {
				s.closeSubscriber(campaignID, sub)
				goto nextSubscriber
			}
		}
	nextSubscriber:
	}
}

func (s *journal) campaignSubscribersLocked(campaignID string) []*subscriptionHandle {
	byCampaign, ok := s.subscribers[campaignID]
	if !ok {
		return nil
	}
	out := make([]*subscriptionHandle, 0, len(byCampaign))
	for handle := range byCampaign {
		out = append(out, handle)
	}
	return out
}

func (s *journal) closeSubscriber(campaignID string, handle *subscriptionHandle) {
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
	handle.CloseRecords()
}
