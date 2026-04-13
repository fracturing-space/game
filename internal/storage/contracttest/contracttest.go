package contracttest

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/event"
	"github.com/fracturing-space/game/internal/participant"
	"github.com/fracturing-space/game/internal/scene"
	"github.com/fracturing-space/game/internal/service"
)

var fixedTime = time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)

func RunJournal(t *testing.T, newJournal func(*testing.T) service.Journal) {
	t.Helper()

	t.Run("append visibility", func(t *testing.T) {
		t.Parallel()

		journal := newJournal(t)
		appended, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
			Events: []event.Envelope{mustEnvelope(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{
				Name: "Autumn Twilight",
			})},
		}}, fixedNow(fixedTime))
		if err != nil {
			t.Fatalf("AppendCommits() error = %v", err)
		}
		if got, want := len(appended), 1; got != want {
			t.Fatalf("appended len = %d, want %d", got, want)
		}

		listed, ok, err := journal.List(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if !ok || len(listed) != 1 || listed[0].Seq != 1 {
			t.Fatalf("List() = (%v,%t), want one visible record at seq 1", listed, ok)
		}

		after, ok, err := journal.ListAfter(context.Background(), "camp-1", 1)
		if err != nil {
			t.Fatalf("ListAfter() error = %v", err)
		}
		if !ok || len(after) != 0 {
			t.Fatalf("ListAfter(1) = (%v,%t), want empty visible tail", after, ok)
		}

		head, ok, err := journal.HeadSeq(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("HeadSeq() error = %v", err)
		}
		if !ok || head != 1 {
			t.Fatalf("HeadSeq() = (%d,%t), want (1,true)", head, ok)
		}
	})

	t.Run("subscribe after checkpoint emits only newer records", func(t *testing.T) {
		t.Parallel()

		journal := newJournal(t)
		if _, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{
			{Events: []event.Envelope{mustEnvelope(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{Name: "Autumn Twilight"})}},
			{Events: []event.Envelope{mustEnvelope(t, participant.JoinedEventSpec, "camp-1", participant.Joined{
				ParticipantID: "part-1",
				Name:          "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
			})}},
		}, fixedNow(fixedTime)); err != nil {
			t.Fatalf("AppendCommits(seed) error = %v", err)
		}

		subscription, err := journal.SubscribeAfter(context.Background(), "camp-1", 1)
		if err != nil {
			t.Fatalf("SubscribeAfter() error = %v", err)
		}
		defer subscription.Close()

		if _, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
			Events: []event.Envelope{mustEnvelope(t, campaign.UpdatedEventSpec, "camp-1", campaign.Updated{Name: "Autumn Eclipse"})},
		}}, fixedNow(fixedTime.Add(time.Minute))); err != nil {
			t.Fatalf("AppendCommits(live) error = %v", err)
		}

		records := receiveRecords(t, subscription.Records, 2)
		if got := sequences(records); !slices.Equal(got, []uint64{2, 3}) {
			t.Fatalf("subscription seqs = %v, want [2 3]", got)
		}
	})

	t.Run("concurrent same campaign appends preserve one total order", func(t *testing.T) {
		t.Parallel()

		journal := newJournal(t)
		const writers = 8
		start := make(chan struct{})
		errs := make(chan error, writers)
		var wg sync.WaitGroup
		for i := range writers {
			wg.Go(func() {
				<-start
				_, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
					Events: []event.Envelope{mustEnvelope(t, campaign.UpdatedEventSpec, "camp-1", campaign.Updated{
						Name: fmt.Sprintf("Campaign %d", i),
					})},
				}}, fixedNow(fixedTime.Add(time.Duration(i)*time.Second)))
				errs <- err
			})
		}
		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("AppendCommits(concurrent) error = %v", err)
			}
		}

		listed, ok, err := journal.List(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if !ok || len(listed) != writers {
			t.Fatalf("List() = (%d,%t), want (%d,true)", len(listed), ok, writers)
		}
		for i, record := range listed {
			want := uint64(i + 1)
			if record.Seq != want {
				t.Fatalf("record[%d].Seq = %d, want %d", i, record.Seq, want)
			}
		}

		head, ok, err := journal.HeadSeq(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("HeadSeq() error = %v", err)
		}
		if !ok || head != writers {
			t.Fatalf("HeadSeq() = (%d,%t), want (%d,true)", head, ok, writers)
		}
	})

	t.Run("concurrent append and subscribe do not miss or duplicate records", func(t *testing.T) {
		t.Parallel()

		journal := newJournal(t)
		if _, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
			Events: []event.Envelope{mustEnvelope(t, campaign.CreatedEventSpec, "camp-1", campaign.Created{Name: "Autumn Twilight"})},
		}}, fixedNow(fixedTime)); err != nil {
			t.Fatalf("AppendCommits(seed) error = %v", err)
		}

		subscription, err := journal.SubscribeAfter(context.Background(), "camp-1", 1)
		if err != nil {
			t.Fatalf("SubscribeAfter() error = %v", err)
		}
		defer subscription.Close()

		const writers = 8
		start := make(chan struct{})
		errs := make(chan error, writers)
		var wg sync.WaitGroup
		for i := range writers {
			wg.Go(func() {
				<-start
				_, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
					Events: []event.Envelope{mustEnvelope(t, campaign.UpdatedEventSpec, "camp-1", campaign.Updated{
						Name: fmt.Sprintf("Update %d", i),
					})},
				}}, fixedNow(fixedTime.Add(time.Duration(i+1)*time.Second)))
				errs <- err
			})
		}
		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("AppendCommits(concurrent) error = %v", err)
			}
		}

		records := receiveRecords(t, subscription.Records, writers)
		seen := make(map[uint64]struct{}, len(records))
		for _, record := range records {
			if record.Seq <= 1 || record.Seq > writers+1 {
				t.Fatalf("subscription record seq = %d, want in [2,%d]", record.Seq, writers+1)
			}
			if _, ok := seen[record.Seq]; ok {
				t.Fatalf("subscription duplicated seq %d", record.Seq)
			}
			seen[record.Seq] = struct{}{}
		}
		want := make([]uint64, 0, writers)
		for i := 2; i <= writers+1; i++ {
			want = append(want, uint64(i))
		}
		if got := slices.Sorted(maps.Keys(seen)); !slices.Equal(got, want) {
			t.Fatalf("subscription seqs = %v, want %v", got, want)
		}
	})

	t.Run("subscription records are clone safe", func(t *testing.T) {
		t.Parallel()

		journal := newJournal(t)
		subscription, err := journal.SubscribeAfter(context.Background(), "camp-1", 0)
		if err != nil {
			t.Fatalf("SubscribeAfter() error = %v", err)
		}
		defer subscription.Close()

		if _, err := journal.AppendCommits(context.Background(), "camp-1", []service.PreparedCommit{{
			Events: []event.Envelope{mustEnvelope(t, scene.CreatedEventSpec, "camp-1", scene.Created{
				SceneID:      "scene-1",
				SessionID:    "sess-1",
				Name:         "Opening",
				CharacterIDs: []string{"char-1"},
			})},
		}}, fixedNow(fixedTime)); err != nil {
			t.Fatalf("AppendCommits() error = %v", err)
		}

		record := <-subscription.Records
		created, err := event.MessageAs[scene.Created](record.Envelope)
		if err != nil {
			t.Fatalf("MessageAs(scene.Created) error = %v", err)
		}
		created.CharacterIDs[0] = "mutated"

		listed, ok, err := journal.List(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if !ok || len(listed) != 1 {
			t.Fatalf("List() = (%d,%t), want (1,true)", len(listed), ok)
		}
		stored, err := event.MessageAs[scene.Created](listed[0].Envelope)
		if err != nil {
			t.Fatalf("MessageAs(stored scene.Created) error = %v", err)
		}
		if got, want := stored.CharacterIDs[0], "char-1"; got != want {
			t.Fatalf("stored character id = %q, want %q", got, want)
		}
	})

	t.Run("subscription close closes records channel", func(t *testing.T) {
		t.Parallel()

		journal := newJournal(t)
		subscription, err := journal.SubscribeAfter(context.Background(), "camp-1", 0)
		if err != nil {
			t.Fatalf("SubscribeAfter() error = %v", err)
		}
		subscription.Close()

		select {
		case _, ok := <-subscription.Records:
			if ok {
				t.Fatal("subscription.Records should be closed after Close()")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("subscription.Records did not close after Close()")
		}
	})
}

func RunProjectionStore(t *testing.T, newStore func(*testing.T) service.ProjectionStore) {
	t.Helper()

	t.Run("returned projections are clone safe", func(t *testing.T) {
		t.Parallel()

		store := newStore(t)
		state := projectionState("camp-1", "Autumn Twilight")
		if err := store.SaveProjection(context.Background(), service.ProjectionSnapshot{
			CampaignID:     "camp-1",
			HeadSeq:        3,
			State:          state,
			UpdatedAt:      fixedTime,
			LastActivityAt: fixedTime,
		}); err != nil {
			t.Fatalf("SaveProjection() error = %v", err)
		}
		if err := store.SaveWatermark(context.Background(), service.ProjectionWatermark{
			CampaignID:      "camp-1",
			AppliedSeq:      3,
			ExpectedNextSeq: 4,
			UpdatedAt:       fixedTime,
		}); err != nil {
			t.Fatalf("SaveWatermark() error = %v", err)
		}

		state.Name = "mutated after save"
		snapshot, ok, err := store.GetProjection(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("GetProjection() error = %v", err)
		}
		if !ok || snapshot.State.Name != "Autumn Twilight" {
			t.Fatalf("GetProjection() = (%q,%t), want stored clone", snapshot.State.Name, ok)
		}
		snapshot.State.Name = "mutated after load"

		again, ok, err := store.GetProjection(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("GetProjection(second) error = %v", err)
		}
		if !ok || again.State.Name != "Autumn Twilight" {
			t.Fatalf("GetProjection(second) = (%q,%t), want stored clone", again.State.Name, ok)
		}

		items, err := store.ListCampaignsBySubject(context.Background(), "subject-1", 10)
		if err != nil {
			t.Fatalf("ListCampaignsBySubject() error = %v", err)
		}
		if got, want := len(items), 1; got != want {
			t.Fatalf("ListCampaignsBySubject() len = %d, want %d", got, want)
		}
	})

	t.Run("concurrent save and get remain consistent", func(t *testing.T) {
		t.Parallel()

		store := newStore(t)
		const iterations = 32
		errs := make(chan error, iterations*2)
		var wg sync.WaitGroup

		for i := range iterations {
			wg.Go(func() {
				name := "Alpha"
				if i%2 == 1 {
					name = "Beta"
				}
				errs <- store.SaveProjection(context.Background(), service.ProjectionSnapshot{
					CampaignID:     "camp-1",
					HeadSeq:        uint64(i + 1),
					State:          projectionState("camp-1", name),
					UpdatedAt:      fixedTime.Add(time.Duration(i) * time.Second),
					LastActivityAt: fixedTime.Add(time.Duration(i) * time.Second),
				})
			})

			wg.Go(func() {
				snapshot, ok, err := store.GetProjection(context.Background(), "camp-1")
				if err != nil {
					errs <- err
					return
				}
				if ok && snapshot.State.Name != "Alpha" && snapshot.State.Name != "Beta" {
					errs <- fmt.Errorf("GetProjection() returned unexpected name %q", snapshot.State.Name)
					return
				}
				errs <- nil
			})
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("concurrent projection operation error = %v", err)
			}
		}

		snapshot, ok, err := store.GetProjection(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("GetProjection(final) error = %v", err)
		}
		if !ok {
			t.Fatal("GetProjection(final) = missing, want stored projection")
		}
		if snapshot.State.Name != "Alpha" && snapshot.State.Name != "Beta" {
			t.Fatalf("GetProjection(final) name = %q, want Alpha or Beta", snapshot.State.Name)
		}
	})

	t.Run("combined projection save rejects mismatched campaigns without partial state", func(t *testing.T) {
		t.Parallel()

		store := newStore(t)
		err := store.SaveProjectionAndWatermark(context.Background(),
			service.ProjectionSnapshot{
				CampaignID:     "camp-1",
				HeadSeq:        3,
				State:          projectionState("camp-1", "Autumn Twilight"),
				UpdatedAt:      fixedTime,
				LastActivityAt: fixedTime,
			},
			service.ProjectionWatermark{
				CampaignID:      "camp-2",
				AppliedSeq:      3,
				ExpectedNextSeq: 4,
				UpdatedAt:       fixedTime,
			},
		)
		if err == nil {
			t.Fatal("SaveProjectionAndWatermark(mismatched campaigns) error = nil, want failure")
		}

		if _, ok, err := store.GetProjection(context.Background(), "camp-1"); err != nil || ok {
			t.Fatalf("GetProjection(camp-1) = (%t,%v), want missing nil", ok, err)
		}
		if _, ok, err := store.GetProjection(context.Background(), "camp-2"); err != nil || ok {
			t.Fatalf("GetProjection(camp-2) = (%t,%v), want missing nil", ok, err)
		}
		if _, ok, err := store.GetWatermark(context.Background(), "camp-1"); err != nil || ok {
			t.Fatalf("GetWatermark(camp-1) = (%t,%v), want missing nil", ok, err)
		}
		if _, ok, err := store.GetWatermark(context.Background(), "camp-2"); err != nil || ok {
			t.Fatalf("GetWatermark(camp-2) = (%t,%v), want missing nil", ok, err)
		}
	})
}

func RunArtifactStore(t *testing.T, newStore func(*testing.T) service.ArtifactStore) {
	t.Helper()

	t.Run("returned artifacts are caller safe", func(t *testing.T) {
		t.Parallel()

		store := newStore(t)
		if err := store.PutArtifact(context.Background(), service.Artifact{
			CampaignID: "camp-1",
			Path:       "story.md",
			Content:    "once upon a time",
			UpdatedAt:  fixedTime,
		}); err != nil {
			t.Fatalf("PutArtifact() error = %v", err)
		}

		item, ok, err := store.GetArtifact(context.Background(), "camp-1", "story.md")
		if err != nil {
			t.Fatalf("GetArtifact() error = %v", err)
		}
		if !ok || item.Content != "once upon a time" {
			t.Fatalf("GetArtifact() = (%+v,%t), want stored artifact", item, ok)
		}
		item.Content = "mutated"

		again, ok, err := store.GetArtifact(context.Background(), "camp-1", "story.md")
		if err != nil {
			t.Fatalf("GetArtifact(second) error = %v", err)
		}
		if !ok || again.Content != "once upon a time" {
			t.Fatalf("GetArtifact(second) = (%+v,%t), want stored artifact", again, ok)
		}

		items, err := store.ListArtifacts(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("ListArtifacts() error = %v", err)
		}
		if got, want := len(items), 1; got != want {
			t.Fatalf("ListArtifacts() len = %d, want %d", got, want)
		}
	})

	t.Run("concurrent put and read remain stable", func(t *testing.T) {
		t.Parallel()

		store := newStore(t)
		const writers = 16
		errs := make(chan error, writers*3)
		var wg sync.WaitGroup

		for i := range writers {
			path := fmt.Sprintf("story-%02d.md", i)
			wg.Go(func() {
				errs <- store.PutArtifact(context.Background(), service.Artifact{
					CampaignID: "camp-1",
					Path:       path,
					Content:    fmt.Sprintf("story %d", i),
					UpdatedAt:  fixedTime.Add(time.Duration(i) * time.Second),
				})
			})

			wg.Go(func() {
				_, _, err := store.GetArtifact(context.Background(), "camp-1", path)
				errs <- err
			})

			wg.Go(func() {
				_, err := store.ListArtifacts(context.Background(), "camp-1")
				errs <- err
			})
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("concurrent artifact operation error = %v", err)
			}
		}

		items, err := store.ListArtifacts(context.Background(), "camp-1")
		if err != nil {
			t.Fatalf("ListArtifacts(final) error = %v", err)
		}
		if got, want := len(items), writers; got != want {
			t.Fatalf("ListArtifacts(final) len = %d, want %d", got, want)
		}
	})
}

func fixedNow(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

func mustEnvelope[T event.Message](t *testing.T, spec event.TypedSpec[T], campaignID string, payload T) event.Envelope {
	t.Helper()

	envelope, err := event.NewEnvelope(spec, campaignID, payload)
	if err != nil {
		t.Fatalf("event.NewEnvelope(%s) error = %v", spec.Definition().Type, err)
	}
	return envelope
}

func receiveRecords(t *testing.T, ch <-chan event.Record, count int) []event.Record {
	t.Helper()

	records := make([]event.Record, 0, count)
	timeout := time.After(2 * time.Second)
	for len(records) < count {
		select {
		case record, ok := <-ch:
			if !ok {
				t.Fatalf("subscription closed early after %d of %d records", len(records), count)
			}
			records = append(records, record)
		case <-timeout:
			t.Fatalf("timed out waiting for %d subscription records", count)
		}
	}
	return records
}

func sequences(records []event.Record) []uint64 {
	out := make([]uint64, 0, len(records))
	for _, record := range records {
		out = append(out, record.Seq)
	}
	return out
}

func projectionState(campaignID string, name string) campaign.State {
	state := campaign.NewState()
	state.Exists = true
	state.CampaignID = campaignID
	state.Name = name
	state.Participants["owner-1"] = participant.Record{
		ID:   "owner-1",
		Name: "Owner", Access: participant.AccessOwner, SubjectID: "subject-1",
		Active: true,
	}
	return state
}
