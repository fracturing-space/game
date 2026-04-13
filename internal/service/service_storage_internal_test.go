package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestReplayLocked(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		state, err := svc.replayLocked([]event.Record{
			{Seq: 1, CommitSeq: 1, Envelope: mustTestEventEnvelope(t, "camp-1", "start")},
			{Seq: 2, CommitSeq: 2, Envelope: mustTestEventEnvelope(t, "camp-1", "next")},
		})
		if err != nil {
			t.Fatalf("replayLocked() error = %v", err)
		}
		if got, want := state.CampaignID, "camp-1"; got != want {
			t.Fatalf("campaign id = %q, want %q", got, want)
		}
		if got, want := state.Name, "next"; got != want {
			t.Fatalf("state name = %q, want %q", got, want)
		}
	})

	t.Run("fold failure", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{
			fold: func(*campaign.State, event.Envelope) error {
				return errors.New("boom")
			},
		})
		if _, err := svc.replayLocked([]event.Record{{Seq: 1, CommitSeq: 1, Envelope: mustTestEventEnvelope(t, "camp-1", "start")}}); err == nil {
			t.Fatal("replayLocked() error = nil, want failure")
		}
	})
}

func TestListEventsAndStoreHelpers(t *testing.T) {
	t.Parallel()

	store := newTestMemoryStore()
	if _, ok, err := store.List(context.Background(), "camp-1"); err != nil {
		t.Fatalf("List() error = %v", err)
	} else if ok {
		t.Fatal("List() should report missing timelines")
	}
	if _, ok, err := store.ListAfter(context.Background(), "camp-1", 0); err != nil {
		t.Fatalf("ListAfter() error = %v", err)
	} else if ok {
		t.Fatal("ListAfter() should report missing timelines")
	}
	if _, ok, err := store.HeadSeq(context.Background(), "camp-1"); err != nil {
		t.Fatalf("HeadSeq() error = %v", err)
	} else if ok {
		t.Fatal("HeadSeq() should report missing timelines")
	}

	first, err := store.AppendCommits(context.Background(), "camp-1", []PreparedCommit{{Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "start")}}}, func() time.Time {
		return serviceTestClockTime
	})
	if err != nil {
		t.Fatalf("AppendCommits(first) error = %v", err)
	}
	second, err := store.AppendCommits(context.Background(), "camp-1", []PreparedCommit{
		{Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "next"), mustTestEventEnvelope(t, "camp-1", "last")}},
	}, func() time.Time {
		return serviceTestClockTime.Add(time.Minute)
	})
	if err != nil {
		t.Fatalf("AppendCommits(second) error = %v", err)
	}
	if got, want := first[0].CommitSeq, uint64(1); got != want {
		t.Fatalf("first commit seq = %d, want %d", got, want)
	}
	if got, want := second[0].Seq, uint64(2); got != want {
		t.Fatalf("second first seq = %d, want %d", got, want)
	}
	if got, want := second[0].CommitSeq, uint64(2); got != want {
		t.Fatalf("second commit seq = %d, want %d", got, want)
	}
	if got, want := second[1].CommitSeq, uint64(2); got != want {
		t.Fatalf("second second commit seq = %d, want %d", got, want)
	}
	if got, want := second[0].RecordedAt, serviceTestClockTime.Add(time.Minute); !got.Equal(want) {
		t.Fatalf("recorded at = %v, want %v", got, want)
	}

	first[0].Seq = 99
	timeline, ok, err := store.List(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !ok {
		t.Fatal("List() should return stored timeline")
	}
	if got, want := timeline[0].Seq, uint64(1); got != want {
		t.Fatalf("stored seq = %d, want %d", got, want)
	}
	after, ok, err := store.ListAfter(context.Background(), "camp-1", 1)
	if err != nil {
		t.Fatalf("ListAfter() error = %v", err)
	}
	if !ok {
		t.Fatal("ListAfter() should return stored timeline")
	}
	if got, want := len(after), 2; got != want {
		t.Fatalf("after len = %d, want %d", got, want)
	}
	if got, want := after[0].Seq, uint64(2); got != want {
		t.Fatalf("after first seq = %d, want %d", got, want)
	}

	if got, ok, err := store.HeadSeq(context.Background(), "camp-1"); err != nil {
		t.Fatalf("HeadSeq() error = %v", err)
	} else if !ok || got != 3 {
		t.Fatalf("HeadSeq() = (%d,%t), want (3,true)", got, ok)
	}

	store.timelines["empty"] = nil
	if got, ok, err := store.HeadSeq(context.Background(), "empty"); err != nil {
		t.Fatalf("HeadSeq(empty) error = %v", err)
	} else if !ok || got != 0 {
		t.Fatalf("HeadSeq(empty) = (%d,%t), want (0,true)", got, ok)
	}
}

func TestListEventsAndCloneHelpers(t *testing.T) {
	t.Parallel()

	svc := newStubService(t, stubServiceModule{})
	if _, err := svc.store.AppendCommits(context.Background(), "camp-1", []PreparedCommit{
		{Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "start")}},
		{Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "next")}},
	}, func() time.Time {
		return serviceTestClockTime
	}); err != nil {
		t.Fatalf("AppendCommits() error = %v", err)
	}
	seedAuthorizedCampaign(t, svc, "camp-1", defaultCaller())
	records, err := svc.ListEvents(context.Background(), defaultCaller(), "camp-1", 2)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if got, want := len(records), 2; got != want {
		t.Fatalf("records len = %d, want %d", got, want)
	}
	records[0].Seq = 99
	again, err := svc.ListEvents(context.Background(), defaultCaller(), "camp-1", 0)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if got, want := again[0].Seq, uint64(1); got != want {
		t.Fatalf("stored seq after mutation = %d, want %d", got, want)
	}

	sourceEnvelopes := []event.Envelope{{
		CampaignID: "camp-1",
		Message: scene.Created{
			SceneID:      "scene-1",
			SessionID:    "sess-1",
			Name:         "Opening Scene",
			CharacterIDs: []string{"char-1"},
		},
	}}
	envelopes := cloneEnvelopes(sourceEnvelopes)
	envelopes[0].CampaignID = "mutated"
	created := envelopes[0].Message.(scene.Created)
	created.CharacterIDs[0] = "char-x"
	envelopes[0].Message = created
	if got, want := sourceEnvelopes[0].CampaignID, "camp-1"; got != want {
		t.Fatalf("source campaign id = %q, want %q", got, want)
	}
	if got, want := sourceEnvelopes[0].Message.(scene.Created).CharacterIDs[0], "char-1"; got != want {
		t.Fatalf("source scene cast = %q, want %q", got, want)
	}

	sourceCommits := []PreparedCommit{{Events: []event.Envelope{{
		CampaignID: "camp-1",
		Message: session.Started{
			SessionID: "sess-1",
			Name:      "Session 1",
			CharacterControllers: []session.CharacterControllerAssignment{{
				CharacterID:   "char-1",
				ParticipantID: "part-1",
			}},
		},
	}}}}
	commits := clonePreparedCommits(sourceCommits)
	commits[0].Events[0].CampaignID = "mutated"
	started := commits[0].Events[0].Message.(session.Started)
	started.CharacterControllers[0].ParticipantID = "part-x"
	commits[0].Events[0].Message = started
	if got, want := commits[0].Events[0].CampaignID, "mutated"; got != want {
		t.Fatalf("mutated campaign id = %q, want %q", got, want)
	}
	if got, want := sourceCommits[0].Events[0].Message.(session.Started).CharacterControllers[0].ParticipantID, "part-1"; got != want {
		t.Fatalf("source session participant id = %q, want %q", got, want)
	}

	sourceRecords := []event.Record{{
		Seq:       1,
		CommitSeq: 1,
		Envelope: event.Envelope{
			CampaignID: "camp-1",
			Message: scene.CastReplaced{
				SceneID:      "scene-1",
				CharacterIDs: []string{"char-1"},
			},
		},
	}}
	clonedRecords := cloneEventRecords(sourceRecords)
	replaced := clonedRecords[0].Message.(scene.CastReplaced)
	replaced.CharacterIDs[0] = "char-x"
	clonedRecords[0].Message = replaced
	if got, want := sourceRecords[0].Message.(scene.CastReplaced).CharacterIDs[0], "char-1"; got != want {
		t.Fatalf("source cast replaced character = %q, want %q", got, want)
	}
}

func TestIDAllocatorAndSession(t *testing.T) {
	t.Parallel()

	allocator := newSequentialIDAllocator()
	allocator.next["camp"] = 1

	session := allocator.Session(true)
	sequentialSession, ok := session.(*sequentialIDSession)
	if !ok {
		t.Fatalf("Session() type = %T, want *sequentialIDSession", session)
	}
	if got, want := sequentialSession.next["camp"], uint64(1); got != want {
		t.Fatalf("session copy = %d, want %d", got, want)
	}

	id, err := session.NewID(" camp ")
	if err != nil {
		t.Fatalf("NewID() error = %v", err)
	}
	if got, want := id, "camp-2"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if _, err := session.NewID("   "); err == nil {
		t.Fatal("NewID() should reject blank prefixes")
	}

	var nilSession *sequentialIDSession
	nilSession.Commit()

	noCommit := allocator.Session(false)
	if _, err := noCommit.NewID("camp"); err != nil {
		t.Fatalf("NewID() error = %v", err)
	}
	noCommit.Commit()
	if got, want := allocator.next["camp"], uint64(1); got != want {
		t.Fatalf("allocator next after non-commit = %d, want %d", got, want)
	}

	(&sequentialIDSession{commit: true}).Commit()
	session.Commit()
	if got, want := allocator.next["camp"], uint64(2); got != want {
		t.Fatalf("allocator next after commit = %d, want %d", got, want)
	}

	opaque := NewOpaqueIDAllocator()
	opaqueSession := opaque.Session(true)
	firstID, err := opaqueSession.NewID("camp")
	if err != nil {
		t.Fatalf("opaque NewID(first) error = %v", err)
	}
	secondID, err := opaqueSession.NewID("camp")
	if err != nil {
		t.Fatalf("opaque NewID(second) error = %v", err)
	}
	if firstID == secondID {
		t.Fatalf("opaque allocator produced duplicate ids: %q", firstID)
	}
	if !strings.HasPrefix(firstID, "camp-") {
		t.Fatalf("opaque id prefix = %q, want camp-*", firstID)
	}
	if _, err := opaqueSession.NewID("   "); err == nil {
		t.Fatal("opaque NewID(blank) error = nil, want failure")
	}
	opaqueSession.Commit()
}

func TestCloneZeroCases(t *testing.T) {
	t.Parallel()

	if got := cloneEnvelopes(nil); got != nil {
		t.Fatalf("cloneEnvelopes(nil) = %v, want nil", got)
	}
	if got := cloneEventRecords(nil); got != nil {
		t.Fatalf("cloneEventRecords(nil) = %v, want nil", got)
	}
	if got := clonePreparedCommits(nil); got != nil {
		t.Fatalf("clonePreparedCommits(nil) = %v, want nil", got)
	}
}

func TestProjectionHelpersAdditionalBranches(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	svc.runtimeTTL = 0
	slot := svc.campaignSlot("camp-1")
	slot.mu.Lock()
	slot.storeRuntime(1, campaign.NewState(), fixedRecordTime.Add(-time.Hour))
	slot.mu.Unlock()
	slot.mu.Lock()
	_, ok := slot.loadRuntime(1, fixedRecordTime, svc.runtimeTTL)
	slot.mu.Unlock()
	if !ok {
		t.Fatal("loadRuntime(ttl disabled) evicted runtime, want hit")
	}

	snapshotTime := fixedRecordTime.Add(time.Minute)
	watermarkTime := fixedRecordTime
	if got, want := projectionUpdatedAt(
		ProjectionSnapshot{UpdatedAt: snapshotTime},
		ProjectionWatermark{UpdatedAt: watermarkTime},
	), snapshotTime; got != want {
		t.Fatalf("projectionUpdatedAt(snapshot newer) = %v, want %v", got, want)
	}
}
