package storeutil

import (
	"context"
	"sync"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
)

// SubscriptionHandle manages one clone-safe per-campaign event subscription.
type SubscriptionHandle struct {
	records     chan event.Record
	live        chan event.Record
	ctx         context.Context
	stop        chan struct{}
	closeOnce   sync.Once
	recordsOnce sync.Once
	closeFn     func()
}

// NewSubscriptionHandle returns one live subscription handle with catch-up
// records starting after afterSeq.
func NewSubscriptionHandle(ctx context.Context, afterSeq uint64, initial []event.Record) *SubscriptionHandle {
	if ctx == nil {
		ctx = context.Background()
	}
	handle := &SubscriptionHandle{
		records: make(chan event.Record, 32),
		live:    make(chan event.Record, 32),
		ctx:     ctx,
		stop:    make(chan struct{}),
	}
	go handle.run(afterSeq, initial)
	return handle
}

// SetCloseFunc registers one callback to run exactly once before the records
// channel is closed.
func (h *SubscriptionHandle) SetCloseFunc(fn func()) {
	if h == nil {
		return
	}
	h.closeFn = fn
}

// Subscription exposes the service-level stream wrapper for the handle.
func (h *SubscriptionHandle) Subscription() service.EventSubscription {
	if h == nil {
		return service.EventSubscription{
			Records: nil,
			Close:   func() {},
		}
	}
	return service.EventSubscription{
		Records: h.records,
		Close: func() {
			h.closeOnce.Do(func() {
				if h.closeFn != nil {
					h.closeFn()
				}
				h.CloseRecords()
			})
		},
	}
}

// Enqueue offers one new live record to the subscription.
func (h *SubscriptionHandle) Enqueue(record event.Record) bool {
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

// CloseRecords stops the subscription and closes the records channel.
func (h *SubscriptionHandle) CloseRecords() {
	if h == nil || h.stop == nil {
		return
	}
	h.recordsOnce.Do(func() {
		close(h.stop)
	})
}

func (h *SubscriptionHandle) run(afterSeq uint64, initial []event.Record) {
	if h == nil || h.records == nil {
		return
	}
	defer close(h.records)

	lastSeq := afterSeq
	for _, record := range initial {
		if record.Seq <= lastSeq {
			continue
		}
		record = CloneRecord(record)
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
			record = CloneRecord(record)
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

// CloneRecord deep-clones one event record so mutable payloads never alias
// storage-owned state.
func CloneRecord(record event.Record) event.Record {
	return event.Record{
		Seq:        record.Seq,
		CommitSeq:  record.CommitSeq,
		RecordedAt: record.RecordedAt,
		Envelope: event.Envelope{
			CampaignID: record.CampaignID,
			Message:    cloneMessage(record.Message),
		},
	}
}

func cloneMessage(message event.Message) event.Message {
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
