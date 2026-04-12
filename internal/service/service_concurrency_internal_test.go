package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/errs"
	"github.com/fracturing-space/game/internal/event"
)

func TestExecuteOnOneCampaignDoesNotBlockInspectOnAnother(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	campA := mustCreateCampaign(t, svc, "Alpha")
	campB := mustCreateCampaign(t, svc, "Beta")

	blocking := newBlockingJournal(svc.store, campA)
	svc.store = blocking

	doneExecute := make(chan error, 1)
	go func() {
		_, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
			CampaignID: campA,
			Message:    campaign.Update{Name: "Alpha Prime"},
		})
		doneExecute <- err
	}()

	blocking.waitEntered(t)

	doneInspect := make(chan error, 1)
	go func() {
		_, err := svc.Inspect(context.Background(), defaultCaller(), campB)
		doneInspect <- err
	}()

	select {
	case err := <-doneInspect:
		if err != nil {
			t.Fatalf("Inspect(other campaign) error = %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Inspect(other campaign) blocked behind unrelated Execute")
	}

	blocking.release()

	if err := <-doneExecute; err != nil {
		t.Fatalf("CommitCommand(blocked campaign) error = %v", err)
	}
}

func TestExecuteDoesNotBlockInspectOnSameCampaign(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	campaignID := mustCreateCampaign(t, svc, "Alpha")

	blocking := newBlockingJournal(svc.store, campaignID)
	svc.store = blocking

	doneExecute := make(chan error, 1)
	go func() {
		_, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
			CampaignID: campaignID,
			Message:    campaign.Update{Name: "Alpha Prime"},
		})
		doneExecute <- err
	}()

	blocking.waitEntered(t)

	doneInspect := make(chan error, 1)
	go func() {
		_, err := svc.Inspect(context.Background(), defaultCaller(), campaignID)
		doneInspect <- err
	}()

	select {
	case err := <-doneInspect:
		if err != nil {
			t.Fatalf("Inspect(same campaign) error = %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Inspect(same campaign) blocked behind committed write path")
	}

	blocking.release()

	if err := <-doneExecute; err != nil {
		t.Fatalf("CommitCommand(blocked campaign) error = %v", err)
	}
}

func TestExecuteDoesNotBlockCommittedSnapshotReadsOnSameCampaign(t *testing.T) {
	t.Parallel()

	fixture := newCreatedCampaignFixture(t)
	blocking := newBlockingJournal(fixture.Service.store, fixture.CampaignID)
	fixture.Service.store = blocking

	doneExecute := make(chan error, 1)
	go func() {
		_, err := fixture.Service.CommitCommand(context.Background(), fixture.OwnerCaller, command.Envelope{
			CampaignID: fixture.CampaignID,
			Message:    campaign.AIBind{AIAgentID: "agent-7"},
		})
		doneExecute <- err
	}()

	blocking.waitEntered(t)

	doneReadiness := make(chan error, 1)
	go func() {
		report, err := fixture.Service.GetPlayReadiness(context.Background(), fixture.OwnerCaller, fixture.CampaignID)
		if err == nil && report.Ready() {
			err = errors.New("GetPlayReadiness() returned ready while write was in flight")
		}
		doneReadiness <- err
	}()

	select {
	case err := <-doneReadiness:
		if err != nil {
			t.Fatalf("GetPlayReadiness() error = %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("GetPlayReadiness() blocked behind committed write path")
	}

	doneListEvents := make(chan error, 1)
	go func() {
		records, err := fixture.Service.ListEvents(context.Background(), fixture.OwnerCaller, fixture.CampaignID, 0)
		if err == nil && len(records) != 2 {
			err = errors.New("ListEvents() returned in-flight event")
		}
		doneListEvents <- err
	}()

	select {
	case err := <-doneListEvents:
		if err != nil {
			t.Fatalf("ListEvents() error = %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ListEvents() blocked behind committed write path")
	}

	doneReadResource := make(chan error, 1)
	go func() {
		resource, err := fixture.Service.ReadResource(context.Background(), fixture.OwnerCaller, "campaign://"+fixture.CampaignID)
		if err == nil && strings.Contains(resource, `"ai_agent_id":"agent-7"`) {
			err = errors.New("ReadResource() returned in-flight state")
		}
		doneReadResource <- err
	}()

	select {
	case err := <-doneReadResource:
		if err != nil {
			t.Fatalf("ReadResource() error = %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ReadResource() blocked behind committed write path")
	}

	blocking.release()

	if err := <-doneExecute; err != nil {
		t.Fatalf("CommitCommand(blocked campaign) error = %v", err)
	}
}

func TestConcurrentCreateRejectsCollidingCampaignID(t *testing.T) {
	t.Parallel()

	manifest, err := BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	svc, err := New(Config{
		Manifest:        manifest,
		IDs:             fixedSequenceAllocator{ids: []string{"camp-1", "part-1", "part-2"}},
		RecordClock:     fixedClock{at: serviceTestClockTime},
		Journal:         newTestMemoryStore(),
		ProjectionStore: newTestProjectionStore(),
		ArtifactStore:   newTestArtifactStore(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			<-start
			_, err := svc.CommitCommand(context.Background(), caller.MustNewSubject("subject-1"), command.Envelope{
				Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
			})
			results <- err
		})
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errs.Is(err, errs.KindAlreadyExists):
			conflicts++
		default:
			t.Fatalf("CommitCommand(concurrent create) error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("create results = (%d success, %d already exists), want (1,1)", successes, conflicts)
	}
}

func mustCreateCampaign(t *testing.T, svc *Service, name string) string {
	t.Helper()

	result, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: name, OwnerName: "louis"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create %q) error = %v", name, err)
	}
	return result.State.ID
}

type blockingJournal struct {
	base       Journal
	campaignID string
	entered    chan struct{}
	releaseCh  chan struct{}
	once       sync.Once
}

func newBlockingJournal(base Journal, campaignID string) *blockingJournal {
	return &blockingJournal{
		base:       base,
		campaignID: campaignID,
		entered:    make(chan struct{}),
		releaseCh:  make(chan struct{}),
	}
}

func (j *blockingJournal) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-j.entered:
	case <-time.After(time.Second):
		t.Fatal("blocking journal never entered AppendCommits")
	}
}

func (j *blockingJournal) release() {
	close(j.releaseCh)
}

func (j *blockingJournal) AppendCommits(ctx context.Context, campaignID string, commits []PreparedCommit, now func() time.Time) ([]event.Record, error) {
	if campaignID == j.campaignID {
		j.once.Do(func() {
			close(j.entered)
		})
		select {
		case <-j.releaseCh:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return j.base.AppendCommits(ctx, campaignID, commits, now)
}

func (j *blockingJournal) List(ctx context.Context, campaignID string) ([]event.Record, bool, error) {
	return j.base.List(ctx, campaignID)
}

func (j *blockingJournal) ListAfter(ctx context.Context, campaignID string, afterSeq uint64) ([]event.Record, bool, error) {
	return j.base.ListAfter(ctx, campaignID, afterSeq)
}

func (j *blockingJournal) HeadSeq(ctx context.Context, campaignID string) (uint64, bool, error) {
	return j.base.HeadSeq(ctx, campaignID)
}

func (j *blockingJournal) SubscribeAfter(ctx context.Context, campaignID string, afterSeq uint64) (EventSubscription, error) {
	return j.base.SubscribeAfter(ctx, campaignID, afterSeq)
}

type fixedSequenceAllocator struct {
	ids []string
}

func (a fixedSequenceAllocator) Session(commit bool) IDSession {
	_ = commit
	return &fixedSequenceSession{
		next: append([]string(nil), a.ids...),
	}
}

type fixedSequenceSession struct {
	index int
	next  []string
}

func (s *fixedSequenceSession) NewID(prefix string) (string, error) {
	if strings.TrimSpace(prefix) == "" {
		return "", errors.New("id prefix is required")
	}
	if s.index >= len(s.next) {
		return "", errors.New("fixed id sequence exhausted")
	}
	next := s.next[s.index]
	s.index++
	return next, nil
}

func (*fixedSequenceSession) Commit() {}
