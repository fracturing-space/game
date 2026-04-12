package memory

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
	"github.com/fracturing-space/game/internal/session"
)

var storeTestTime = time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)

func TestBundleProvidesAllPorts(t *testing.T) {
	t.Parallel()

	bundle := NewBundle()
	if bundle.Journal == nil {
		t.Fatal("Journal = nil")
	}
	if bundle.ProjectionStore == nil {
		t.Fatal("ProjectionStore = nil")
	}
	if bundle.ArtifactStore == nil {
		t.Fatal("ArtifactStore = nil")
	}
}

func TestJournalStoresAndStreamsRecords(t *testing.T) {
	t.Parallel()

	journal := NewJournal()
	subscription, err := journal.SubscribeAfter(context.Background(), "camp-1", 0)
	if err != nil {
		t.Fatalf("SubscribeAfter() error = %v", err)
	}
	defer subscription.Close()

	first, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
		Events: []event.Envelope{{
			CampaignID: "camp-1",
		}},
	}}, func() time.Time {
		return storeTestTime
	})
	if err != nil {
		t.Fatalf("AppendCommits(first) error = %v", err)
	}
	second, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
		Events: []event.Envelope{{CampaignID: "camp-1"}},
	}}, func() time.Time {
		return storeTestTime.Add(time.Minute)
	})
	if err != nil {
		t.Fatalf("AppendCommits(second) error = %v", err)
	}
	if got, want := first[0].Seq, uint64(1); got != want {
		t.Fatalf("first seq = %d, want %d", got, want)
	}
	if got, want := second[0].Seq, uint64(2); got != want {
		t.Fatalf("second seq = %d, want %d", got, want)
	}

	timeline, ok, err := journal.List(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !ok || len(timeline) != 2 {
		t.Fatalf("List() = (%d,%t), want (2,true)", len(timeline), ok)
	}
	after, ok, err := journal.ListAfter(context.Background(), "camp-1", 1)
	if err != nil {
		t.Fatalf("ListAfter() error = %v", err)
	}
	if !ok || len(after) != 1 || after[0].Seq != 2 {
		t.Fatalf("ListAfter() = (%v,%t), want seq 2", after, ok)
	}
	head, ok, err := journal.HeadSeq(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("HeadSeq() error = %v", err)
	}
	if !ok || head != 2 {
		t.Fatalf("HeadSeq() = (%d,%t), want (2,true)", head, ok)
	}

	streamed := []uint64{(<-subscription.Records).Seq, (<-subscription.Records).Seq}
	if got, want := streamed, []uint64{1, 2}; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("streamed seqs = %v, want %v", got, want)
	}
}

func TestJournalClosesSlowSubscriberOnOverflow(t *testing.T) {
	t.Parallel()

	journal := NewJournal()
	subscription, err := journal.SubscribeAfter(context.Background(), "camp-1", 0)
	if err != nil {
		t.Fatalf("SubscribeAfter() error = %v", err)
	}
	defer subscription.Close()

	events := make([]event.Envelope, 33)
	for i := range events {
		events[i] = event.Envelope{CampaignID: "camp-1"}
	}

	if _, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{Events: events}}, func() time.Time {
		return storeTestTime
	}); err != nil {
		t.Fatalf("AppendCommits() error = %v", err)
	}

	count := 0
	for range subscription.Records {
		count++
	}
	if count >= len(events) {
		t.Fatalf("delivered records = %d, want slow-subscriber close before all %d records", count, len(events))
	}
}

func TestProjectionStoreClonesState(t *testing.T) {
	t.Parallel()

	store := NewProjectionStore()
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "stored"

	if err := store.SaveProjectionAndWatermark(context.Background(), service.ProjectionSnapshot{
		CampaignID:     "camp-1",
		HeadSeq:        3,
		State:          state,
		UpdatedAt:      storeTestTime,
		LastActivityAt: storeTestTime,
	}, service.ProjectionWatermark{
		CampaignID:      "camp-1",
		AppliedSeq:      3,
		ExpectedNextSeq: 4,
		UpdatedAt:       storeTestTime.Add(time.Minute),
	}); err != nil {
		t.Fatalf("SaveProjectionAndWatermark() error = %v", err)
	}

	state.Name = "mutated"
	snapshot, ok, err := store.GetProjection(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetProjection() error = %v", err)
	}
	if !ok {
		t.Fatal("GetProjection() = missing, want stored projection")
	}
	if got, want := snapshot.State.Name, "stored"; got != want {
		t.Fatalf("snapshot state name = %q, want %q", got, want)
	}
	if got, want := snapshot.LastActivityAt, storeTestTime; !got.Equal(want) {
		t.Fatalf("snapshot last activity at = %v, want %v", got, want)
	}

	snapshot.State.Name = "mutated after load"
	again, ok, err := store.GetProjection(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetProjection(second) error = %v", err)
	}
	if !ok {
		t.Fatal("GetProjection(second) = missing, want stored projection")
	}
	if got, want := again.State.Name, "stored"; got != want {
		t.Fatalf("snapshot clone name = %q, want %q", got, want)
	}

	watermark, ok, err := store.GetWatermark(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetWatermark() error = %v", err)
	}
	if !ok || watermark.AppliedSeq != 3 {
		t.Fatalf("GetWatermark() = (%+v,%t), want applied seq 3", watermark, ok)
	}

	items, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("ListCampaignsBySubject() len = %d, want 0 without bound participant", len(items))
	}
}

func TestProjectionStoreListsCampaignsBySubject(t *testing.T) {
	t.Parallel()

	store := NewProjectionStore()

	buildState := func(campaignID string, ready bool) campaign.State {
		state := campaign.NewState()
		state.Exists = true
		state.CampaignID = campaignID
		state.Name = campaignID
		state.Participants["owner-1"] = participant.Record{
			ID:   "owner-1",
			Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
			Active: true,
		}
		if ready {
			state.AIAgentID = "agent-1"
			state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
			state.ActiveSessionID = "sess-1"
			state.ActiveSceneID = "scene-1"
			state.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening Scene", Active: true}
			state.Characters["char-1"] = character.Record{
				ID:            "char-1",
				ParticipantID: "owner-1",
				Name:          "Luna", Active: true,
			}
		}
		return state
	}

	if err := store.SaveProjection(context.Background(), service.ProjectionSnapshot{
		CampaignID:     "camp-1",
		HeadSeq:        1,
		State:          buildState("camp-1", false),
		UpdatedAt:      storeTestTime,
		LastActivityAt: storeTestTime,
	}); err != nil {
		t.Fatalf("SaveProjection(camp-1) error = %v", err)
	}
	if err := store.SaveProjection(context.Background(), service.ProjectionSnapshot{
		CampaignID:     "camp-2",
		HeadSeq:        2,
		State:          buildState("camp-2", true),
		UpdatedAt:      storeTestTime.Add(time.Minute),
		LastActivityAt: storeTestTime.Add(time.Minute),
	}); err != nil {
		t.Fatalf("SaveProjection(camp-2) error = %v", err)
	}

	items, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject() error = %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("ListCampaignsBySubject() len = %d, want %d", got, want)
	}
	if got, want := items[0].CampaignID, "camp-2"; got != want {
		t.Fatalf("ListCampaignsBySubject()[0].CampaignID = %q, want %q", got, want)
	}
	if !items[0].ReadyToPlay {
		t.Fatal("ListCampaignsBySubject()[0].ReadyToPlay = false, want true")
	}
	if got, want := items[1].CampaignID, "camp-1"; got != want {
		t.Fatalf("ListCampaignsBySubject()[1].CampaignID = %q, want %q", got, want)
	}

	if _, err := store.ListCampaignsBySubject(context.Background(), "   ", 10); err == nil {
		t.Fatal("ListCampaignsBySubject(blank) error = nil, want failure")
	}
	if none, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 0); err != nil {
		t.Fatalf("ListCampaignsBySubject(limit=0) error = %v", err)
	} else if len(none) != 0 {
		t.Fatalf("ListCampaignsBySubject(limit=0) len = %d, want 0", len(none))
	}

	updated := buildState("camp-1", false)
	updated.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-9",
		Active: true,
	}
	if err := store.SaveProjection(context.Background(), service.ProjectionSnapshot{
		CampaignID:     "camp-1",
		HeadSeq:        3,
		State:          updated,
		UpdatedAt:      storeTestTime.Add(2 * time.Minute),
		LastActivityAt: storeTestTime.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("SaveProjection(updated camp-1) error = %v", err)
	}
	items, err = store.ListCampaignsBySubject(context.Background(), "subject-1", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject(after rebind) error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("ListCampaignsBySubject(after rebind) len = %d, want %d", got, want)
	}
	rebound, err := store.ListCampaignsBySubject(context.Background(), "subject-9", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject(subject-9) error = %v", err)
	}
	if got, want := len(rebound), 1; got != want {
		t.Fatalf("ListCampaignsBySubject(subject-9) len = %d, want %d", got, want)
	}
	if got, want := rebound[0].CampaignID, "camp-1"; got != want {
		t.Fatalf("ListCampaignsBySubject(subject-9)[0].CampaignID = %q, want %q", got, want)
	}

	updatedReady := buildState("camp-2", true)
	updatedReady.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-9",
		Active: true,
	}
	if err := store.SaveProjection(context.Background(), service.ProjectionSnapshot{
		CampaignID:     "camp-2",
		HeadSeq:        4,
		State:          updatedReady,
		UpdatedAt:      storeTestTime.Add(3 * time.Minute),
		LastActivityAt: storeTestTime.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("SaveProjection(updated camp-2) error = %v", err)
	}

	emptied, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject(subject-1 cleared) error = %v", err)
	}
	if len(emptied) != 0 {
		t.Fatalf("ListCampaignsBySubject(subject-1 cleared) len = %d, want 0", len(emptied))
	}
	rebound, err = store.ListCampaignsBySubject(context.Background(), "subject-9", 10)
	if err != nil {
		t.Fatalf("ListCampaignsBySubject(subject-9 after second rebind) error = %v", err)
	}
	if got, want := len(rebound), 2; got != want {
		t.Fatalf("ListCampaignsBySubject(subject-9 after second rebind) len = %d, want %d", got, want)
	}
}

func TestStoresSupportConcurrentAccess(t *testing.T) {
	t.Parallel()

	projections := NewProjectionStore()
	artifacts := NewArtifactStore()

	state := func(campaignID string, subjectID string) campaign.State {
		next := campaign.NewState()
		next.Exists = true
		next.CampaignID = campaignID
		next.Name = campaignID
		next.Participants["owner-1"] = participant.Record{
			ID:   "owner-1",
			Name: "Owner", Access: participant.AccessOwner, SubjectID: subjectID,
			Active: true,
		}
		return next
	}

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for i := range 100 {
			if err := projections.SaveProjection(context.Background(), service.ProjectionSnapshot{
				CampaignID:     "camp-1",
				HeadSeq:        uint64(i + 1),
				State:          state("camp-1", "subject-1"),
				UpdatedAt:      storeTestTime.Add(time.Duration(i) * time.Second),
				LastActivityAt: storeTestTime.Add(time.Duration(i) * time.Second),
			}); err != nil {
				t.Errorf("SaveProjection(camp-1) error = %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := range 100 {
			if _, err := projections.ListCampaignsBySubject(context.Background(), "subject-1", 10); err != nil {
				t.Errorf("ListCampaignsBySubject() error = %v", err)
				return
			}
			if _, _, err := projections.GetProjection(context.Background(), "camp-1"); err != nil {
				t.Errorf("GetProjection() error = %v", err)
				return
			}
			if _, _, err := projections.GetWatermark(context.Background(), "camp-1"); err != nil {
				t.Errorf("GetWatermark() error = %v", err)
				return
			}
			if err := projections.SaveWatermark(context.Background(), service.ProjectionWatermark{
				CampaignID:      "camp-1",
				AppliedSeq:      uint64(i),
				ExpectedNextSeq: uint64(i + 1),
				UpdatedAt:       storeTestTime.Add(time.Duration(i) * time.Second),
			}); err != nil {
				t.Errorf("SaveWatermark() error = %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := range 100 {
			if err := artifacts.PutArtifact(context.Background(), service.Artifact{
				CampaignID: "camp-1",
				Path:       "story.md",
				Content:    "version",
				UpdatedAt:  storeTestTime.Add(time.Duration(i) * time.Second),
			}); err != nil {
				t.Errorf("PutArtifact() error = %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for range 100 {
			if _, _, err := artifacts.GetArtifact(context.Background(), "camp-1", "story.md"); err != nil {
				t.Errorf("GetArtifact() error = %v", err)
				return
			}
			if _, err := artifacts.ListArtifacts(context.Background(), "camp-1"); err != nil {
				t.Errorf("ListArtifacts() error = %v", err)
				return
			}
		}
	}()

	wg.Wait()
}

func TestCampaignSummaryHelpers(t *testing.T) {
	t.Parallel()

	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = "camp-1"
	state.Name = "Autumn Twilight"
	state.AIAgentID = "agent-1"
	state.Sessions["sess-1"] = session.Record{ID: "sess-1", Name: "Session 1", Status: session.StatusActive}
	state.ActiveSessionID = "sess-1"
	state.ActiveSceneID = "scene-1"
	state.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
		Active: true,
	}
	state.Participants["dupe"] = participant.Record{
		ID:   "dupe",
		Name: "Owner 2", Access: participant.AccessMember, SubjectID: "subject-1",
		Active: true,
	}
	state.Scenes["scene-1"] = scene.Record{ID: "scene-1", SessionID: "sess-1", Name: "Opening Scene", Active: true}
	state.Characters["char-1"] = character.Record{ID: "char-1", ParticipantID: "owner-1", Name: "Luna", Active: true}

	summary := service.CampaignSummaryFromSnapshot(service.ProjectionSnapshot{
		CampaignID:     "camp-1",
		HeadSeq:        1,
		State:          state,
		UpdatedAt:      storeTestTime,
		LastActivityAt: storeTestTime,
	})
	if !summary.HasAIBinding {
		t.Fatal("campaignSummaryFromSnapshot().HasAIBinding = false, want true")
	}
	if !summary.HasActiveSession {
		t.Fatal("campaignSummaryFromSnapshot().HasActiveSession = false, want true")
	}
	if summary.ReadyToPlay {
		t.Fatal("campaignSummaryFromSnapshot().ReadyToPlay = true, want false")
	}

	subjects := service.BoundSubjectIDs(state)
	if got, want := len(subjects), 1; got != want {
		t.Fatalf("boundSubjectIDs() len = %d, want %d", got, want)
	}

	items := []service.CampaignSummary{
		{CampaignID: "camp-b", LastActivityAt: storeTestTime},
		{CampaignID: "camp-a", LastActivityAt: storeTestTime},
	}
	slices.SortFunc(items, service.CompareCampaignSummary)
	if got, want := []string{items[0].CampaignID, items[1].CampaignID}, []string{"camp-a", "camp-b"}; !slices.Equal(got, want) {
		t.Fatalf("compareCampaignSummary() order = %v, want %v", got, want)
	}
}

func TestArtifactStoreRejectsNonCanonicalPaths(t *testing.T) {
	t.Parallel()

	store := NewArtifactStore()
	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: "camp-1",
		Path:       "/story.md",
		Content:    "# Harbor",
		UpdatedAt:  storeTestTime,
	}); err == nil {
		t.Fatal("PutArtifact(non-canonical path) error = nil, want failure")
	}

	if err := store.PutArtifact(context.Background(), service.Artifact{
		CampaignID: "camp-1",
		Path:       "story.md",
		Content:    "# Harbor",
		UpdatedAt:  storeTestTime,
	}); err != nil {
		t.Fatalf("PutArtifact(canonical) error = %v", err)
	}

	item, ok, err := store.GetArtifact(context.Background(), "camp-1", "story.md")
	if err != nil {
		t.Fatalf("GetArtifact() error = %v", err)
	}
	if !ok || item.Content != "# Harbor" {
		t.Fatalf("GetArtifact() = (%+v,%t), want stored artifact", item, ok)
	}

	items, err := store.ListArtifacts(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("ListArtifacts() error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("ListArtifacts() len = %d, want %d", got, want)
	}
}
