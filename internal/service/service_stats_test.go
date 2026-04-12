package service

import (
	"context"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/command"
)

func TestServiceStatsReturnsZeroForNilService(t *testing.T) {
	t.Parallel()

	if got := (*Service)(nil).Stats(); got != (Stats{}) {
		t.Fatalf("nil service Stats() = %+v, want zero value", got)
	}

	svc := &Service{}
	if got := svc.Stats(); got != (Stats{}) {
		t.Fatalf("service without slots Stats() = %+v, want zero value", got)
	}
}

func TestServiceStatsTrackSlotsSnapshotsAndRuntimes(t *testing.T) {
	t.Parallel()

	clock := &mutableClock{at: fixedRecordTime}
	svc := newTestService(t)
	svc.recordClock = clock
	svc.runtimeTTL = 10 * time.Minute

	if got := svc.Stats(); got != (Stats{}) {
		t.Fatalf("initial Stats() = %+v, want zero value", got)
	}

	svc.campaignSlot("camp-empty")
	stats := svc.Stats()
	if got, want := stats.CampaignSlots, 1; got != want {
		t.Fatalf("Stats().CampaignSlots = %d, want %d", got, want)
	}
	if stats.PublishedSnapshots != 0 || stats.WarmRuntimes != 0 {
		t.Fatalf("empty slot stats = %+v, want no published snapshots or runtimes", stats)
	}

	first, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "Alpha", OwnerName: "louis"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create first) error = %v", err)
	}

	stats = svc.Stats()
	if got, want := stats.CampaignSlots, 2; got != want {
		t.Fatalf("Stats().CampaignSlots after first create = %d, want %d", got, want)
	}
	if got, want := stats.PublishedSnapshots, 1; got != want {
		t.Fatalf("Stats().PublishedSnapshots after first create = %d, want %d", got, want)
	}
	if got, want := stats.WarmRuntimes, 1; got != want {
		t.Fatalf("Stats().WarmRuntimes after first create = %d, want %d", got, want)
	}
	if got, want := stats.OldestPublishedAt, fixedRecordTime; !got.Equal(want) {
		t.Fatalf("Stats().OldestPublishedAt = %v, want %v", got, want)
	}
	if got, want := stats.OldestRuntimeTouchedAt, fixedRecordTime; !got.Equal(want) {
		t.Fatalf("Stats().OldestRuntimeTouchedAt = %v, want %v", got, want)
	}

	clock.at = clock.at.Add(time.Minute)
	if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "Beta", OwnerName: "louis"},
	}); err != nil {
		t.Fatalf("CommitCommand(create second) error = %v", err)
	}

	stats = svc.Stats()
	if got, want := stats.CampaignSlots, 3; got != want {
		t.Fatalf("Stats().CampaignSlots after second create = %d, want %d", got, want)
	}
	if got, want := stats.PublishedSnapshots, 2; got != want {
		t.Fatalf("Stats().PublishedSnapshots after second create = %d, want %d", got, want)
	}
	if got, want := stats.WarmRuntimes, 2; got != want {
		t.Fatalf("Stats().WarmRuntimes after second create = %d, want %d", got, want)
	}
	if got, want := stats.OldestPublishedAt, fixedRecordTime; !got.Equal(want) {
		t.Fatalf("Stats().OldestPublishedAt after second create = %v, want %v", got, want)
	}
	if got, want := stats.OldestRuntimeTouchedAt, fixedRecordTime; !got.Equal(want) {
		t.Fatalf("Stats().OldestRuntimeTouchedAt after second create = %v, want %v", got, want)
	}

	if _, err := svc.Inspect(context.Background(), defaultCaller(), first.State.ID); err != nil {
		t.Fatalf("Inspect(first campaign) error = %v", err)
	}
}

func TestServiceStatsExcludeExpiredRuntimesWithoutDroppingPublishedSnapshots(t *testing.T) {
	t.Parallel()

	clock := &mutableClock{at: fixedRecordTime}
	svc := newTestService(t)
	svc.recordClock = clock
	svc.runtimeTTL = time.Minute

	if _, err := svc.CommitCommand(context.Background(), defaultCaller(), command.Envelope{
		Message: campaign.Create{Name: "Alpha", OwnerName: "louis"},
	}); err != nil {
		t.Fatalf("CommitCommand(create) error = %v", err)
	}

	stats := svc.Stats()
	if got, want := stats.WarmRuntimes, 1; got != want {
		t.Fatalf("Stats().WarmRuntimes before expiry = %d, want %d", got, want)
	}

	clock.at = clock.at.Add(2 * time.Minute)
	stats = svc.Stats()
	if got, want := stats.PublishedSnapshots, 1; got != want {
		t.Fatalf("Stats().PublishedSnapshots after expiry = %d, want %d", got, want)
	}
	if got, want := stats.WarmRuntimes, 0; got != want {
		t.Fatalf("Stats().WarmRuntimes after expiry = %d, want %d", got, want)
	}
	if !stats.OldestPublishedAt.Equal(fixedRecordTime) {
		t.Fatalf("Stats().OldestPublishedAt after expiry = %v, want %v", stats.OldestPublishedAt, fixedRecordTime)
	}
	if !stats.OldestRuntimeTouchedAt.IsZero() {
		t.Fatalf("Stats().OldestRuntimeTouchedAt after expiry = %v, want zero", stats.OldestRuntimeTouchedAt)
	}
}

func TestServiceStatsHandleNilSlotsAndNilClock(t *testing.T) {
	t.Parallel()

	slot := &campaignSlot{}
	slot.storePublished(3, campaign.State{
		Exists:     true,
		CampaignID: "camp-1",
		Name:       "Alpha",
	}, fixedRecordTime)
	slot.storeRuntime(3, campaign.State{
		Exists:     true,
		CampaignID: "camp-1",
		Name:       "Alpha",
	}, fixedRecordTime.Add(time.Minute))

	svc := &Service{
		slots: &campaignSlotRegistry{
			slots: map[string]*campaignSlot{
				"camp-nil":  nil,
				"camp-live": slot,
			},
		},
		runtimeTTL: time.Minute,
	}

	stats := svc.Stats()
	if got, want := stats.CampaignSlots, 2; got != want {
		t.Fatalf("Stats().CampaignSlots = %d, want %d", got, want)
	}
	if got, want := stats.PublishedSnapshots, 1; got != want {
		t.Fatalf("Stats().PublishedSnapshots = %d, want %d", got, want)
	}
	if got, want := stats.WarmRuntimes, 1; got != want {
		t.Fatalf("Stats().WarmRuntimes = %d, want %d", got, want)
	}
	if !stats.OldestPublishedAt.Equal(fixedRecordTime) {
		t.Fatalf("Stats().OldestPublishedAt = %v, want %v", stats.OldestPublishedAt, fixedRecordTime)
	}
	if !stats.OldestRuntimeTouchedAt.Equal(fixedRecordTime.Add(time.Minute)) {
		t.Fatalf("Stats().OldestRuntimeTouchedAt = %v, want %v", stats.OldestRuntimeTouchedAt, fixedRecordTime.Add(time.Minute))
	}
}
