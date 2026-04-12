package storeutil

import (
	"context"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestSubscriptionHandleClonesAndCloses(t *testing.T) {
	t.Parallel()

	if got := ((*SubscriptionHandle)(nil)).Subscription(); got.Close == nil || got.Records != nil {
		t.Fatalf("nil Subscription() = %+v, want inert close-only subscription", got)
	}

	var zero SubscriptionHandle
	zero.CloseRecords()

	base := event.Record{
		Seq:        2,
		CommitSeq:  1,
		RecordedAt: time.Unix(0, 0).UTC(),
		Envelope: event.Envelope{
			CampaignID: "camp-1",
			Message: scene.Created{
				SceneID:      "scene-9",
				CharacterIDs: []string{"char-9"},
			},
		},
	}
	cloned := CloneRecord(base)
	clonedScene := cloned.Message.(scene.Created)
	clonedScene.CharacterIDs[0] = "changed"
	originalScene := base.Message.(scene.Created)
	if got, want := originalScene.CharacterIDs[0], "char-9"; got != want {
		t.Fatalf("CloneRecord() mutated original = %q, want %q", got, want)
	}
}

func TestSubscriptionHandleRunAndEnqueueBranches(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	handle := NewSubscriptionHandle(ctx, 1, []event.Record{
		{Seq: 1, Envelope: event.Envelope{CampaignID: "camp-1"}},
		{Seq: 2, Envelope: event.Envelope{CampaignID: "camp-1"}},
	})
	if handle == nil {
		t.Fatal("NewSubscriptionHandle() = nil")
	}
	defer handle.Subscription().Close()

	record := <-handle.records
	if got, want := record.Seq, uint64(2); got != want {
		t.Fatalf("initial streamed seq = %d, want %d", got, want)
	}
	if !handle.Enqueue(event.Record{Seq: 3, Envelope: event.Envelope{CampaignID: "camp-1"}}) {
		t.Fatal("Enqueue(open) = false, want true")
	}
	if got, want := (<-handle.records).Seq, uint64(3); got != want {
		t.Fatalf("live streamed seq = %d, want %d", got, want)
	}

	handle.CloseRecords()
	if handle.Enqueue(event.Record{Seq: 4, Envelope: event.Envelope{CampaignID: "camp-1"}}) {
		t.Fatal("Enqueue(closed) = true, want false")
	}
	if _, ok := <-handle.records; ok {
		t.Fatal("records channel should close after CloseRecords")
	}

	cancel()
}

func TestCloneRecordClonesMutablePayloads(t *testing.T) {
	t.Parallel()

	created := CloneRecord(event.Record{
		Envelope: event.Envelope{
			CampaignID: "camp-1",
			Message:    scene.Created{SceneID: "scene-1", CharacterIDs: []string{"char-1"}},
		},
	}).Message.(scene.Created)
	created.CharacterIDs[0] = "mutated"
	createdAgain := CloneRecord(event.Record{
		Envelope: event.Envelope{
			CampaignID: "camp-1",
			Message:    scene.Created{SceneID: "scene-1", CharacterIDs: []string{"char-1"}},
		},
	}).Message.(scene.Created)
	if got, want := createdAgain.CharacterIDs[0], "char-1"; got != want {
		t.Fatalf("CloneRecord(scene.Created) = %q, want %q", got, want)
	}

	started := CloneRecord(event.Record{
		Envelope: event.Envelope{
			CampaignID: "camp-1",
			Message: session.Started{
				SessionID:            "sess-1",
				CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
			},
		},
	}).Message.(session.Started)
	started.CharacterControllers[0].ParticipantID = "changed"
	startedAgain := CloneRecord(event.Record{
		Envelope: event.Envelope{
			CampaignID: "camp-1",
			Message: session.Started{
				SessionID:            "sess-1",
				CharacterControllers: []session.CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
			},
		},
	}).Message.(session.Started)
	if got, want := startedAgain.CharacterControllers[0].ParticipantID, "part-1"; got != want {
		t.Fatalf("CloneRecord(session.Started) = %q, want %q", got, want)
	}
}
