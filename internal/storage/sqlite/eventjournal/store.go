package eventjournal

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/storage/sqlite/db"
	"github.com/fracturing-space/game/internal/storage/sqlite/internalutil"
	"github.com/fracturing-space/game/internal/storage/sqlite/migrations"
	"github.com/fracturing-space/game/internal/storage/storeutil"
)

// Store implements the service journal port on SQLite.
type Store struct {
	db          *sql.DB
	q           *db.Queries
	codec       *eventEnvelopeCodec
	writeMu     sync.Mutex
	mu          sync.Mutex
	subscribers map[string]map[*subscriptionHandle]struct{}
}

type subscriptionHandle = storeutil.SubscriptionHandle

var (
	openSQLiteDB    = internalutil.Open
	applyMigrations = func(sqlDB *sql.DB) error {
		return internalutil.ApplyMigrations(sqlDB, migrations.EventsFS, "events", time.Now)
	}
)

// Open opens a SQLite event journal store at the provided path.
func Open(path string, catalog *event.Catalog) (*Store, error) {
	codec, err := newEventEnvelopeCodec(catalog)
	if err != nil {
		return nil, err
	}
	sqlDB, err := openSQLiteDB(path)
	if err != nil {
		return nil, fmt.Errorf("open events store: %w", err)
	}
	if err := applyMigrations(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate events store: %w", err)
	}
	return &Store{
		db:          sqlDB,
		q:           db.New(sqlDB),
		codec:       codec,
		subscribers: make(map[string]map[*subscriptionHandle]struct{}),
	}, nil
}

// Close closes the underlying SQLite database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// AppendCommits appends immutable event records for one campaign.
func (s *Store) AppendCommits(ctx context.Context, campaignID string, commits []service.PreparedCommit, now func() time.Time) (records []event.Record, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin append commits transaction for %s: %w", campaignID, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := s.q.WithTx(tx)

	head, err := qtx.GetEventHead(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("load event head for %s: %w", campaignID, err)
	}
	nextSeq := uint64(head.HeadSeq) + 1
	nextCommitSeq := uint64(head.HeadCommitSeq) + 1

	records = make([]event.Record, 0)
	for _, commit := range commits {
		recordedAt := now().UTC()
		recordedAtNS := recordedAt.UnixNano()
		for _, envelope := range commit.Events {
			eventType, payloadBlob, err := s.codec.Encode(envelope)
			if err != nil {
				return nil, fmt.Errorf("encode event %s for %s: %w", envelope.Type(), campaignID, err)
			}
			record := event.Record{
				Seq:        nextSeq,
				CommitSeq:  nextCommitSeq,
				RecordedAt: recordedAt,
				Envelope:   envelope,
			}
			if err := qtx.AppendEvent(ctx, db.AppendEventParams{
				CampaignID:   campaignID,
				Seq:          int64(nextSeq),
				CommitSeq:    int64(nextCommitSeq),
				RecordedAtNs: recordedAtNS,
				EventType:    eventType,
				PayloadBlob:  payloadBlob,
			}); err != nil {
				return nil, fmt.Errorf("append event %s seq %d for %s: %w", envelope.Type(), nextSeq, campaignID, err)
			}
			records = append(records, record)
			nextSeq++
		}
		nextCommitSeq++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit append commits transaction for %s: %w", campaignID, err)
	}

	subscribers := s.campaignSubscribers(campaignID)
	cloned := append([]event.Record(nil), records...)
	s.notifySubscribers(campaignID, subscribers, cloned)
	return cloned, nil
}

// List returns all stored events for one campaign.
func (s *Store) List(ctx context.Context, campaignID string) ([]event.Record, bool, error) {
	return s.listAfter(ctx, campaignID, 0)
}

// ListAfter returns stored events after one sequence for one campaign.
func (s *Store) ListAfter(ctx context.Context, campaignID string, afterSeq uint64) ([]event.Record, bool, error) {
	return s.listAfter(ctx, campaignID, afterSeq)
}

func (s *Store) listAfter(ctx context.Context, campaignID string, afterSeq uint64) ([]event.Record, bool, error) {
	rows, err := s.q.ListEventsAfter(ctx, db.ListEventsAfterParams{
		CampaignID: campaignID,
		Seq:        int64(afterSeq),
	})
	if err != nil {
		return nil, false, fmt.Errorf("list events after seq %d for %s: %w", afterSeq, campaignID, err)
	}
	if len(rows) == 0 {
		head, err := s.q.GetEventHead(ctx, campaignID)
		if err != nil {
			return nil, false, fmt.Errorf("load event head for %s: %w", campaignID, err)
		}
		return nil, head.EventCount > 0, nil
	}
	return s.recordsFromRows(campaignID, rows)
}

// HeadSeq returns the latest event sequence for one campaign.
func (s *Store) HeadSeq(ctx context.Context, campaignID string) (uint64, bool, error) {
	head, err := s.q.GetEventHead(ctx, campaignID)
	if err != nil {
		return 0, false, fmt.Errorf("load event head for %s: %w", campaignID, err)
	}
	if head.EventCount == 0 {
		return 0, false, nil
	}
	return uint64(head.HeadSeq), true, nil
}

// SubscribeAfter returns one in-process tail subscription starting after one
// committed sequence for one campaign.
func (s *Store) SubscribeAfter(ctx context.Context, campaignID string, afterSeq uint64) (service.EventSubscription, error) {
	s.mu.Lock()
	rows, err := s.q.ListEventsAfter(ctx, db.ListEventsAfterParams{
		CampaignID: campaignID,
		Seq:        int64(afterSeq),
	})
	initial, _, decodeErr := s.recordsFromRows(campaignID, rows)
	if err != nil || decodeErr != nil {
		s.mu.Unlock()
		if err != nil {
			return service.EventSubscription{}, fmt.Errorf("load initial subscription events after seq %d for %s: %w", afterSeq, campaignID, err)
		}
		return service.EventSubscription{}, decodeErr
	}

	handle := storeutil.NewSubscriptionHandle(ctx, afterSeq, initial)
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

func (s *Store) notifySubscribers(campaignID string, subscribers []*subscriptionHandle, records []event.Record) {
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

func (s *Store) campaignSubscribers(campaignID string) []*subscriptionHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *Store) closeSubscriber(campaignID string, handle *subscriptionHandle) {
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

func (s *Store) recordsFromRows(campaignID string, rows []db.ListEventsAfterRow) ([]event.Record, bool, error) {
	records := make([]event.Record, 0, len(rows))
	for _, row := range rows {
		envelope, err := s.codec.Decode(campaignID, event.Type(row.EventType), row.PayloadBlob)
		if err != nil {
			return nil, false, fmt.Errorf("decode event %s seq %d for %s: %w", row.EventType, row.Seq, campaignID, err)
		}
		records = append(records, event.Record{
			Seq:        uint64(row.Seq),
			CommitSeq:  uint64(row.CommitSeq),
			RecordedAt: time.Unix(0, row.RecordedAtNs).UTC(),
			Envelope:   envelope,
		})
	}
	return records, true, nil
}
