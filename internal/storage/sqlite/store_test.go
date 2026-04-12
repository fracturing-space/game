package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fracturing-space/game/internal/caller"
	"github.com/fracturing-space/game/internal/campaign"
	"github.com/fracturing-space/game/internal/character"
	"github.com/fracturing-space/game/internal/command"
	"github.com/fracturing-space/game/internal/service"
)

func TestBundlePersistsCampaignStateAcrossRestart(t *testing.T) {
	t.Parallel()

	paths := Paths{
		EventsDBPath:      filepath.Join(t.TempDir(), "events.db"),
		ProjectionsDBPath: filepath.Join(t.TempDir(), "projections.db"),
		ArtifactsDBPath:   filepath.Join(t.TempDir(), "artifacts.db"),
	}
	manifest, err := service.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	owner := caller.MustNewSubject("subject-1")

	stores, err := Open(manifest, paths)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}
	svc, err := service.New(service.Config{
		Manifest:        manifest,
		RecordClock:     fixedClock{at: fixedRecordTime},
		Journal:         stores.Journal,
		ProjectionStore: stores.ProjectionStore,
		ArtifactStore:   stores.ArtifactStore,
	})
	if err != nil {
		t.Fatalf("service.New(first) error = %v", err)
	}

	create, err := svc.CommitCommand(context.Background(), owner, command.Envelope{
		Message: campaign.Create{Name: "Autumn Twilight", OwnerName: "louis"},
	})
	if err != nil {
		t.Fatalf("CommitCommand(create) error = %v", err)
	}
	campaignID := create.State.ID

	if _, err := svc.CommitCommand(context.Background(), owner, command.Envelope{
		CampaignID: campaignID,
		Message:    campaign.AIBind{AIAgentID: "agent-7"},
	}); err != nil {
		t.Fatalf("CommitCommand(ai bind) error = %v", err)
	}
	if _, err := svc.CommitCommand(context.Background(), owner, command.Envelope{
		CampaignID: campaignID,
		Message: character.Create{
			ParticipantID: create.State.Participants[0].ID,
			Name:          "luna"},
	}); err != nil {
		t.Fatalf("CommitCommand(create character) error = %v", err)
	}
	if err := stores.ArtifactStore.PutArtifact(context.Background(), service.Artifact{
		CampaignID: campaignID,
		Path:       "story.md",
		Content:    "# Harbor\nThe bells toll.",
		UpdatedAt:  fixedRecordTime,
	}); err != nil {
		t.Fatalf("PutArtifact() error = %v", err)
	}

	if err := stores.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	stores, err = Open(manifest, paths)
	if err != nil {
		t.Fatalf("Open(second) error = %v", err)
	}
	defer func() {
		if err := stores.Close(); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	}()

	svc, err = service.New(service.Config{
		Manifest:        manifest,
		RecordClock:     fixedClock{at: fixedRecordTime},
		Journal:         stores.Journal,
		ProjectionStore: stores.ProjectionStore,
		ArtifactStore:   stores.ArtifactStore,
	})
	if err != nil {
		t.Fatalf("service.New(second) error = %v", err)
	}

	inspection, err := svc.Inspect(context.Background(), owner, campaignID)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if got, want := inspection.HeadSeq, uint64(4); got != want {
		t.Fatalf("head seq = %d, want %d", got, want)
	}
	if got, want := inspection.State.Name, "Autumn Twilight"; got != want {
		t.Fatalf("campaign name = %q, want %q", got, want)
	}
	if got, want := inspection.State.AIAgentID, "agent-7"; got != want {
		t.Fatalf("ai agent id = %q, want %q", got, want)
	}
	if got, want := len(inspection.State.Characters), 1; got != want {
		t.Fatalf("character count = %d, want %d", got, want)
	}

	projections := stores.ProjectionStore
	projection, ok, err := projections.GetProjection(context.Background(), campaignID)
	if err != nil {
		t.Fatalf("GetProjection() error = %v", err)
	}
	if !ok {
		t.Fatal("GetProjection() = missing, want persisted projection")
	}
	if got, want := projection.HeadSeq, inspection.HeadSeq; got != want {
		t.Fatalf("projection head seq = %d, want %d", got, want)
	}

	watermark, ok, err := projections.GetWatermark(context.Background(), campaignID)
	if err != nil {
		t.Fatalf("GetWatermark() error = %v", err)
	}
	if !ok {
		t.Fatal("GetWatermark() = missing, want persisted watermark")
	}
	if got, want := watermark.AppliedSeq, inspection.HeadSeq; got != want {
		t.Fatalf("watermark applied seq = %d, want %d", got, want)
	}

	artifactText, err := svc.ReadResource(context.Background(), owner, "campaign://"+campaignID+"/artifacts/story.md")
	if err != nil {
		t.Fatalf("ReadResource(artifact) error = %v", err)
	}
	if !strings.Contains(artifactText, "# Harbor") {
		t.Fatalf("artifact content = %q, want markdown content", artifactText)
	}
}

type fixedClock struct {
	at time.Time
}

func (c fixedClock) Now() time.Time {
	return c.at
}

var fixedRecordTime = time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)
