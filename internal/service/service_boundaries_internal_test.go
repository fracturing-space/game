package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/session"
)

func TestNewRequiresProjectionAndArtifactStore(t *testing.T) {
	t.Parallel()

	manifest := mustManifest(t, nil)

	_, err := New(Config{
		Manifest: manifest,
		Journal:  newTestMemoryStore(),
	})
	if err == nil || !strings.Contains(err.Error(), "projection store is required") {
		t.Fatalf("New() error = %v, want missing projection store failure", err)
	}

	_, err = New(Config{
		Manifest:        manifest,
		Journal:         newTestMemoryStore(),
		ProjectionStore: newTestProjectionStore(),
	})
	if err == nil || !strings.Contains(err.Error(), "artifact store is required") {
		t.Fatalf("New() error = %v, want missing artifact store failure", err)
	}
}

func TestNewRejectsIncompleteManifest(t *testing.T) {
	t.Parallel()

	manifest := mustManifest(t, nil)
	manifest.Events = nil

	_, err := New(Config{
		Manifest:        manifest,
		Journal:         newTestMemoryStore(),
		ProjectionStore: newTestProjectionStore(),
		ArtifactStore:   newTestArtifactStore(),
	})
	if err == nil || !strings.Contains(err.Error(), "service manifest is incomplete") {
		t.Fatalf("New() error = %v, want incomplete manifest failure", err)
	}
}

func TestCampaignSlotRegistryHandlesClonesAndEviction(t *testing.T) {
	t.Parallel()

	registry := newCampaignSlotRegistry()
	source := campaign.NewState()
	source.Exists = true
	source.CampaignID = "camp-1"
	source.Name = "stored"

	slot := registry.Slot("camp-1")
	slot.storeRuntime(3, source, fixedRecordTime)
	source.Name = "mutated after store"

	loaded, ok := slot.loadRuntime(3, fixedRecordTime, 5*time.Minute)
	if !ok {
		t.Fatal("Load() = miss, want hit")
	}
	if got, want := loaded.Name, "stored"; got != want {
		t.Fatalf("loaded name = %q, want %q", got, want)
	}

	loaded.Name = "mutated after load"
	loadedAgain, ok := slot.loadRuntime(3, fixedRecordTime, 5*time.Minute)
	if !ok {
		t.Fatal("Load(second) = miss, want hit")
	}
	if got, want := loadedAgain.Name, "stored"; got != want {
		t.Fatalf("loaded clone name = %q, want %q", got, want)
	}

	if got, want := registry.Slot("camp-1"), slot; got != want {
		t.Fatal("Slot(canonical) should return existing slot")
	}
	if got := registry.Slot(" camp-1 "); got == slot {
		t.Fatal("Slot(padded) should not return existing slot")
	}

	freshSlot := registry.Slot("camp-fresh")
	freshSlot.storeRuntime(1, source, fixedRecordTime.Add(4*time.Minute))
	if _, ok := slot.loadRuntime(3, fixedRecordTime.Add(6*time.Minute), 5*time.Minute); ok {
		t.Fatal("Load(evicted) = hit, want miss")
	}
	if _, ok := freshSlot.loadRuntime(1, fixedRecordTime.Add(6*time.Minute), 5*time.Minute); !ok {
		t.Fatal("Load(fresh) = miss, want hit")
	}

	idleSlot := registry.Acquire("camp-idle", fixedRecordTime, 5*time.Minute)
	registry.Release("camp-idle", idleSlot, fixedRecordTime, 5*time.Minute)

	liveSlot := registry.Acquire("camp-live", fixedRecordTime, 5*time.Minute)
	if got := registry.Acquire("camp-live", fixedRecordTime.Add(time.Minute), 5*time.Minute); got != liveSlot {
		t.Fatal("Acquire(live) should return existing slot")
	}
	registry.Release("camp-live", liveSlot, fixedRecordTime.Add(time.Minute), 5*time.Minute)

	registry.Acquire("camp-sweep", fixedRecordTime.Add(6*time.Minute), 5*time.Minute)
	if _, ok := registry.slots["camp-idle"]; ok {
		t.Fatal("idle slot should be evicted after idle ttl")
	}
	if _, ok := registry.slots["camp-live"]; !ok {
		t.Fatal("live slot should remain while leased")
	}
	registry.Release("camp-live", liveSlot, fixedRecordTime.Add(6*time.Minute), 5*time.Minute)

	reacquired := registry.Acquire("camp-live", fixedRecordTime.Add(12*time.Minute), 5*time.Minute)
	if reacquired == liveSlot {
		t.Fatal("Acquire(evicted live slot) should return a fresh slot")
	}
}

func TestLoadCampaignLockedProjectionPathsAndFailures(t *testing.T) {
	t.Parallel()

	t.Run("journal list failure", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		svc.store = failingJournal{
			base:    svc.store,
			listErr: errors.New("journal list failed"),
		}

		if _, _, err := svc.loadCampaignLocked(context.Background(), "camp-1"); err == nil || !strings.Contains(err.Error(), "journal list failed") {
			t.Fatalf("loadCampaignLocked() error = %v, want journal list failure", err)
		}
	})

	t.Run("projection hit skips replay", func(t *testing.T) {
		foldCalls := 0
		svc := newStubService(t, stubServiceModule{
			fold: func(state *campaign.State, envelope event.Envelope) error {
				foldCalls++
				return defaultStubFold(state, envelope)
			},
		})
		if _, err := svc.store.AppendCommits(context.Background(), "camp-1", []PreparedCommit{{
			Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "journal-state")},
		}}, func() time.Time {
			return serviceTestClockTime
		}); err != nil {
			t.Fatalf("AppendCommits() error = %v", err)
		}

		projectionState := campaign.NewState()
		projectionState.Exists = true
		projectionState.CampaignID = "camp-1"
		projectionState.Name = "projection-state"
		if err := svc.projections.SaveProjection(context.Background(), ProjectionSnapshot{
			CampaignID: "camp-1",
			HeadSeq:    1,
			State:      projectionState,
			UpdatedAt:  serviceTestClockTime,
		}); err != nil {
			t.Fatalf("SaveProjection() error = %v", err)
		}
		_, state, err := svc.loadCampaignLocked(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("loadCampaignLocked() error = %v", err)
		}
		if got, want := state.Name, "projection-state"; got != want {
			t.Fatalf("state name = %q, want %q", got, want)
		}
		if got := foldCalls; got != 0 {
			t.Fatalf("fold calls = %d, want projection reuse without replay", got)
		}
	})

	t.Run("projection load failure", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		if _, err := svc.store.AppendCommits(context.Background(), "camp-1", []PreparedCommit{{
			Events: []event.Envelope{mustTestEventEnvelope(t, "camp-1", "start")},
		}}, func() time.Time {
			return serviceTestClockTime
		}); err != nil {
			t.Fatalf("AppendCommits() error = %v", err)
		}
		svc.projections = failingProjectionStore{
			base:   svc.projections,
			getErr: errors.New("projection load failed"),
		}

		if _, _, err := svc.loadCampaignLocked(context.Background(), "camp-1"); err == nil || !strings.Contains(err.Error(), "projection load failed") {
			t.Fatalf("loadCampaignLocked() error = %v, want projection load failure", err)
		}
	})
}

func TestPersistProjectionLockedErrors(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"

	t.Run("projection save failure", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		svc.projections = failingProjectionStore{
			base:              svc.projections,
			saveProjectionErr: errors.New("projection save failed"),
		}

		if err := svc.persistProjectionLocked(context.Background(), "camp-1", state, 2, fixedRecordTime); err == nil || !strings.Contains(err.Error(), "projection save failed") {
			t.Fatalf("persistProjectionLocked() error = %v, want projection save failure", err)
		}
	})

	t.Run("watermark save failure", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		svc.projections = failingProjectionStore{
			base:             svc.projections,
			saveWatermarkErr: errors.New("watermark save failed"),
		}

		if err := svc.persistProjectionLocked(context.Background(), "camp-1", state, 2, fixedRecordTime); err == nil || !strings.Contains(err.Error(), "watermark save failed") {
			t.Fatalf("persistProjectionLocked() error = %v, want watermark save failure", err)
		}
		if _, ok := testLoadRuntime(svc, "camp-1", 2); ok {
			t.Fatal("runtime cache should not update after watermark save failure")
		}
	})
}

func TestPersistProjectionLockedDefaultsLastActivityAt(t *testing.T) {
	t.Parallel()

	svc := newStubService(t, stubServiceModule{})
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "Autumn Twilight"

	if err := svc.persistProjectionLocked(context.Background(), "camp-1", state, 2, time.Time{}); err != nil {
		t.Fatalf("persistProjectionLocked() error = %v", err)
	}

	projections := svc.projections.(*testProjectionStore)
	snapshot, ok, err := projections.GetProjection(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetProjection() error = %v", err)
	}
	if !ok {
		t.Fatal("GetProjection() = missing, want stored projection")
	}
	if got, want := snapshot.LastActivityAt, serviceTestClockTime; !got.Equal(want) {
		t.Fatalf("snapshot.LastActivityAt = %v, want %v", got, want)
	}
}

func TestSlotWrapperMethods(t *testing.T) {
	t.Parallel()

	t.Run("load campaign wrapper rejects blank id", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		if _, _, err := svc.loadCampaignLocked(context.Background(), "   "); err == nil {
			t.Fatal("loadCampaignLocked(blank) error = nil, want failure")
		}
	})

	t.Run("persist projection wrapper rejects blank id", func(t *testing.T) {
		svc := newStubService(t, stubServiceModule{})
		state := campaign.NewState()
		state.Exists = true
		state.CampaignID = "camp-1"
		if err := svc.persistProjectionLocked(context.Background(), "   ", state, 1, time.Time{}); err == nil {
			t.Fatal("persistProjectionLocked(blank) error = nil, want failure")
		}
	})

	t.Run("read authorized wrapper uses slot path", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		campaignID, timeline, state, err := fixture.Service.readAuthorizedCampaignLocked(context.Background(), fixture.CampaignID, nil)
		if err != nil {
			t.Fatalf("readAuthorizedCampaignLocked() error = %v", err)
		}
		if got, want := campaignID, fixture.CampaignID; got != want {
			t.Fatalf("campaign id = %q, want %q", got, want)
		}
		if len(timeline) == 0 {
			t.Fatal("timeline = empty, want seeded records")
		}
		if got, want := state.CampaignID, fixture.CampaignID; got != want {
			t.Fatalf("state campaign id = %q, want %q", got, want)
		}
	})

}

func TestProjectionHelpers(t *testing.T) {
	t.Parallel()

	if got := headSeqOf(nil); got != 0 {
		t.Fatalf("headSeqOf(nil) = %d, want 0", got)
	}
	if got := headSeqOf([]event.Record{{Seq: 1}, {Seq: 4}}); got != 4 {
		t.Fatalf("headSeqOf(last) = %d, want 4", got)
	}
	if !staleProjectionGap(ProjectionSnapshot{HeadSeq: 2}, 3) {
		t.Fatal("staleProjectionGap() = false, want true")
	}
	if staleProjectionGap(ProjectionSnapshot{HeadSeq: 3}, 3) {
		t.Fatal("staleProjectionGap() = true, want false")
	}
	if got := projectionLag(ProjectionWatermark{AppliedSeq: 5}, 4); got != 0 {
		t.Fatalf("projectionLag(applied ahead) = %d, want 0", got)
	}
	if got := projectionLag(ProjectionWatermark{AppliedSeq: 2}, 5); got != 3 {
		t.Fatalf("projectionLag() = %d, want 3", got)
	}

	snapshotTime := fixedRecordTime
	watermarkTime := fixedRecordTime.Add(time.Minute)
	if got := projectionUpdatedAt(
		ProjectionSnapshot{UpdatedAt: snapshotTime},
		ProjectionWatermark{UpdatedAt: watermarkTime},
	); !got.Equal(watermarkTime) {
		t.Fatalf("projectionUpdatedAt() = %v, want %v", got, watermarkTime)
	}

	timeline := []event.Record{
		{Seq: 1, RecordedAt: fixedRecordTime},
		{Seq: 2, RecordedAt: fixedRecordTime.Add(time.Minute)},
	}
	if got, want := lastActivityAtOfTimeline(timeline), fixedRecordTime.Add(time.Minute); !got.Equal(want) {
		t.Fatalf("lastActivityAtOfTimeline() = %v, want %v", got, want)
	}
	if !lastActivityAtOfTimeline(nil).IsZero() {
		t.Fatal("lastActivityAtOfTimeline(nil) should be zero")
	}

	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "Autumn Twilight"
	state.AIAgentID = "agent-1"
	state.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
		Active: true,
	}
	state.Participants["player-2"] = participant.Record{
		ID:   "player-2",
		Name: "Player 2", Access: participant.AccessMember, SubjectID: "subject-2",
		Active: true,
	}
	state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
	state.ActiveSessionID = "sess-1"
	state.ActiveSceneID = "scene-1"
	state.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening Scene", Active: true}
	state.Characters["char-1"] = character.Record{ID: "char-1", ParticipantID: "owner-1", Name: "Luna", Active: true}
	state.Characters["char-2"] = character.Record{ID: "char-2", ParticipantID: "player-2", Name: "Nova", Active: true}

	summary := CampaignSummaryFromSnapshot(ProjectionSnapshot{
		CampaignID:     "camp-1",
		HeadSeq:        2,
		State:          state,
		UpdatedAt:      fixedRecordTime,
		LastActivityAt: fixedRecordTime.Add(time.Minute),
	})
	if !summary.ReadyToPlay {
		t.Fatal("campaignSummaryFromSnapshot().ReadyToPlay = false, want true")
	}
	if !summary.HasAIBinding {
		t.Fatal("campaignSummaryFromSnapshot().HasAIBinding = false, want true")
	}
	if !summary.HasActiveSession {
		t.Fatal("campaignSummaryFromSnapshot().HasActiveSession = false, want true")
	}

	subjectIDs := BoundSubjectIDs(state)
	if got, want := subjectIDs, []string{"subject-1", "subject-2"}; !slices.Equal(got, want) {
		t.Fatalf("boundSubjectIDs() = %v, want %v", got, want)
	}

	ordered := []CampaignSummary{
		{CampaignID: "camp-b", LastActivityAt: fixedRecordTime},
		{CampaignID: "camp-a", LastActivityAt: fixedRecordTime},
		{CampaignID: "camp-c", LastActivityAt: fixedRecordTime.Add(time.Minute)},
	}
	slices.SortFunc(ordered, CompareCampaignSummary)
	if got, want := []string{ordered[0].CampaignID, ordered[1].CampaignID, ordered[2].CampaignID}, []string{"camp-c", "camp-a", "camp-b"}; !slices.Equal(got, want) {
		t.Fatalf("compareCampaignSummary order = %v, want %v", got, want)
	}
}

func TestReadResourceErrors(t *testing.T) {
	t.Parallel()

	fixture := newActiveSessionCampaignFixture(t)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fixture.Service.ReadResource(canceledCtx, fixture.OwnerCaller, "context://current"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadResource(canceled) error = %v, want context canceled", err)
	}

	tests := []struct {
		name string
		uri  string
		want string
	}{
		{name: "blank uri", uri: "   ", want: "resource uri is required"},
		{name: "unknown scheme", uri: "file://campaign", want: "unknown resource uri"},
		{name: "missing campaign id", uri: "campaign://", want: "campaign id is required"},
		{name: "unknown character shape", uri: "campaign://" + fixture.CampaignID + "/characters/char-1", want: "unknown character resource uri"},
		{name: "missing character", uri: "campaign://" + fixture.CampaignID + "/characters/missing/sheet", want: "character missing not found"},
		{name: "unknown session shape", uri: "campaign://" + fixture.CampaignID + "/sessions/sess-1", want: "unknown session resource uri"},
		{name: "missing session", uri: "campaign://" + fixture.CampaignID + "/sessions/missing/scenes", want: "session missing not found"},
		{name: "artifact path required", uri: "campaign://" + fixture.CampaignID + "/artifacts", want: "artifact path is required"},
		{name: "artifact missing", uri: "campaign://" + fixture.CampaignID + "/artifacts/missing.md", want: "artifact missing.md not found"},
		{name: "unknown campaign segment", uri: "campaign://" + fixture.CampaignID + "/unknown", want: "unknown resource uri"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fixture.Service.ReadResource(context.Background(), fixture.OwnerCaller, tc.uri)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ReadResource(%s) error = %v, want substring %q", tc.uri, err, tc.want)
			}
		})
	}

	fixture.Service.artifacts = failingArtifactStore{
		base:   fixture.Service.artifacts,
		getErr: errors.New("artifact load failed"),
	}
	if _, err := fixture.Service.ReadResource(context.Background(), fixture.OwnerCaller, "campaign://"+fixture.CampaignID+"/artifacts/story.md"); err == nil || !strings.Contains(err.Error(), "artifact load failed") {
		t.Fatalf("ReadResource(artifact error) error = %v, want artifact load failure", err)
	}

	if _, err := marshalResource(make(chan int)); err == nil {
		t.Fatal("marshalResource() error = nil, want marshal failure")
	}
}

func TestSubscribeEventsFailureAndTailBehavior(t *testing.T) {
	t.Parallel()

	t.Run("canceled context", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := fixture.Service.SubscribeEvents(ctx, fixture.OwnerCaller, fixture.CampaignID, 0); !errors.Is(err, context.Canceled) {
			t.Fatalf("SubscribeEvents() error = %v, want context canceled", err)
		}
	})

	t.Run("blank campaign id", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		if _, err := fixture.Service.SubscribeEvents(context.Background(), fixture.OwnerCaller, "   ", 0); err == nil {
			t.Fatal("SubscribeEvents(blank campaign id) error = nil, want failure")
		}
	})

	t.Run("list after failure", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		fixture.Service.store = failingJournal{
			base:              fixture.Service.store,
			subscribeAfterErr: errors.New("tail bootstrap failed"),
		}

		if _, err := fixture.Service.SubscribeEvents(context.Background(), fixture.OwnerCaller, fixture.CampaignID, 0); err == nil || !strings.Contains(err.Error(), "tail bootstrap failed") {
			t.Fatalf("SubscribeEvents() error = %v, want tail bootstrap failure", err)
		}
	})

	t.Run("dedupes repeated sequence and closes on subscription end", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		timeline, ok, err := fixture.Service.store.List(context.Background(), fixture.CampaignID)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if !ok {
			t.Fatal("List() = missing, want seeded campaign")
		}

		subscriptionRecords := make(chan event.Record, 3)
		subscriptionRecords <- event.Record{Seq: 2, CommitSeq: 1, Envelope: timeline[len(timeline)-1].Envelope}
		subscriptionRecords <- event.Record{Seq: 3, CommitSeq: 2, Envelope: timeline[len(timeline)-1].Envelope}
		subscriptionRecords <- event.Record{Seq: 4, CommitSeq: 3, Envelope: timeline[len(timeline)-1].Envelope}
		close(subscriptionRecords)

		fixture.Service.store = failingJournal{
			base: fixture.Service.store,
			subscription: EventSubscription{
				Records: subscriptionRecords,
				Close:   func() {},
			},
		}

		stream, err := fixture.Service.SubscribeEvents(context.Background(), fixture.OwnerCaller, fixture.CampaignID, 1)
		if err != nil {
			t.Fatalf("SubscribeEvents() error = %v", err)
		}
		defer stream.Close()

		var seqs []uint64
		for record := range stream.Records {
			seqs = append(seqs, record.Seq)
		}
		if got, want := seqs, []uint64{2, 3, 4}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
			t.Fatalf("stream seqs = %v, want %v", got, want)
		}
	})

	t.Run("close stops goroutine without context cancellation", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)

		stream, err := fixture.Service.SubscribeEvents(context.Background(), fixture.OwnerCaller, fixture.CampaignID, 99)
		if err != nil {
			t.Fatalf("SubscribeEvents() error = %v", err)
		}
		stream.Close()

		select {
		case _, ok := <-stream.Records:
			if ok {
				t.Fatal("stream records channel should close after Close()")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for stream close")
		}
	})

	t.Run("existing stream keeps tailing after subscriber is unbound", func(t *testing.T) {
		fixture := newCreatedCampaignFixture(t)
		joined, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
			CampaignID: fixture.CampaignID,
			Message: participant.Join{
				Name: "Zoe", Access: participant.AccessMember},
		})
		if err != nil {
			t.Fatalf("CommitCommand(join) error = %v", err)
		}

		seatID := ""
		for _, next := range joined.State.Participants {
			if next.Name == "Zoe" {
				seatID = next.ID
				break
			}
		}
		if seatID == "" {
			t.Fatal("joined participant id = empty, want participant")
		}

		subscriber := callerWithSubject("subject-2")
		if _, err := fixture.Service.CommitCommand(context.Background(), subscriber, command.Envelope{
			CampaignID: fixture.CampaignID,
			Message:    participant.Bind{ParticipantID: seatID, SubjectID: "subject-2"},
		}); err != nil {
			t.Fatalf("CommitCommand(bind) error = %v", err)
		}

		timeline, err := fixture.Service.ListEvents(context.Background(), fixture.OwnerCaller, fixture.CampaignID, 0)
		if err != nil {
			t.Fatalf("ListEvents() error = %v", err)
		}

		stream, err := fixture.Service.SubscribeEvents(context.Background(), subscriber, fixture.CampaignID, headSeqOf(timeline))
		if err != nil {
			t.Fatalf("SubscribeEvents() error = %v", err)
		}
		defer stream.Close()

		if _, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
			CampaignID: fixture.CampaignID,
			Message:    participant.Unbind{ParticipantID: seatID},
		}); err != nil {
			t.Fatalf("CommitCommand(unbind) error = %v", err)
		}
		if _, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
			CampaignID: fixture.CampaignID,
			Message:    campaign.Update{Name: "After Unbind"},
		}); err != nil {
			t.Fatalf("CommitCommand(update) error = %v", err)
		}

		got := make([]event.Type, 0, 2)
		for len(got) < 2 {
			select {
			case record, ok := <-stream.Records:
				if !ok {
					t.Fatalf("stream closed after %d events, want 2", len(got))
				}
				got = append(got, record.Type())
			case <-time.After(time.Second):
				t.Fatalf("timed out waiting for tailed events, got %v", got)
			}
		}
		if want := []event.Type{participant.EventTypeUnbound, campaign.EventTypeUpdated}; !slices.Equal(got, want) {
			t.Fatalf("stream event types = %v, want %v", got, want)
		}
	})
}

type failingJournal struct {
	appendErr         error
	base              Journal
	listErr           error
	listAfterErr      error
	headSeqErr        error
	subscription      EventSubscription
	subscribeAfterErr error
}

func (j failingJournal) AppendCommits(ctx context.Context, campaignID string, commits []PreparedCommit, now func() time.Time) ([]event.Record, error) {
	if j.appendErr != nil {
		return nil, j.appendErr
	}
	if j.base == nil {
		return nil, nil
	}
	return j.base.AppendCommits(ctx, campaignID, commits, now)
}

func (j failingJournal) List(ctx context.Context, campaignID string) ([]event.Record, bool, error) {
	if j.listErr != nil {
		return nil, false, j.listErr
	}
	if j.base == nil {
		return nil, false, nil
	}
	return j.base.List(ctx, campaignID)
}

func (j failingJournal) ListAfter(ctx context.Context, campaignID string, afterSeq uint64) ([]event.Record, bool, error) {
	if j.listAfterErr != nil {
		return nil, false, j.listAfterErr
	}
	if j.base == nil {
		return nil, false, nil
	}
	return j.base.ListAfter(ctx, campaignID, afterSeq)
}

func (j failingJournal) HeadSeq(ctx context.Context, campaignID string) (uint64, bool, error) {
	if j.headSeqErr != nil {
		return 0, false, j.headSeqErr
	}
	if j.base == nil {
		return 0, false, nil
	}
	return j.base.HeadSeq(ctx, campaignID)
}

func (j failingJournal) SubscribeAfter(ctx context.Context, campaignID string, afterSeq uint64) (EventSubscription, error) {
	if j.subscribeAfterErr != nil {
		return EventSubscription{}, j.subscribeAfterErr
	}
	if j.subscription.Records != nil || j.subscription.Close != nil {
		return j.subscription, nil
	}
	if j.base == nil {
		return EventSubscription{Close: func() {}}, nil
	}
	return j.base.SubscribeAfter(ctx, campaignID, afterSeq)
}

type failingProjectionStore struct {
	base              ProjectionStore
	listErr           error
	getErr            error
	saveProjectionErr error
	getWatermarkErr   error
	saveWatermarkErr  error
}

func (s failingProjectionStore) GetProjection(ctx context.Context, campaignID string) (ProjectionSnapshot, bool, error) {
	if s.getErr != nil {
		return ProjectionSnapshot{}, false, s.getErr
	}
	if s.base == nil {
		return ProjectionSnapshot{}, false, nil
	}
	return s.base.GetProjection(ctx, campaignID)
}

func (s failingProjectionStore) SaveProjection(ctx context.Context, snapshot ProjectionSnapshot) error {
	if s.saveProjectionErr != nil {
		return s.saveProjectionErr
	}
	if s.base == nil {
		return nil
	}
	return s.base.SaveProjection(ctx, snapshot)
}

func (s failingProjectionStore) GetWatermark(ctx context.Context, campaignID string) (ProjectionWatermark, bool, error) {
	if s.getWatermarkErr != nil {
		return ProjectionWatermark{}, false, s.getWatermarkErr
	}
	if s.base == nil {
		return ProjectionWatermark{}, false, nil
	}
	return s.base.GetWatermark(ctx, campaignID)
}

func (s failingProjectionStore) SaveWatermark(ctx context.Context, watermark ProjectionWatermark) error {
	if s.saveWatermarkErr != nil {
		return s.saveWatermarkErr
	}
	if s.base == nil {
		return nil
	}
	return s.base.SaveWatermark(ctx, watermark)
}

func (s failingProjectionStore) SaveProjectionAndWatermark(ctx context.Context, snapshot ProjectionSnapshot, watermark ProjectionWatermark) error {
	if s.saveProjectionErr != nil {
		return s.saveProjectionErr
	}
	if s.saveWatermarkErr != nil {
		return s.saveWatermarkErr
	}
	if s.base == nil {
		return nil
	}
	return s.base.SaveProjectionAndWatermark(ctx, snapshot, watermark)
}

func (s failingProjectionStore) ListCampaignsBySubject(ctx context.Context, subjectID string, limit int) ([]CampaignSummary, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.base == nil {
		return nil, nil
	}
	return s.base.ListCampaignsBySubject(ctx, subjectID, limit)
}

type failingArtifactStore struct {
	base   ArtifactStore
	getErr error
}

func (s failingArtifactStore) PutArtifact(ctx context.Context, item Artifact) error {
	if s.base == nil {
		return nil
	}
	return s.base.PutArtifact(ctx, item)
}

func (s failingArtifactStore) GetArtifact(ctx context.Context, campaignID string, path string) (Artifact, bool, error) {
	if s.getErr != nil {
		return Artifact{}, false, s.getErr
	}
	if s.base == nil {
		return Artifact{}, false, nil
	}
	return s.base.GetArtifact(ctx, campaignID, path)
}

func (s failingArtifactStore) ListArtifacts(ctx context.Context, campaignID string) ([]Artifact, error) {
	if s.base == nil {
		return nil, nil
	}
	return s.base.ListArtifacts(ctx, campaignID)
}
