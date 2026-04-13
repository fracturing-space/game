package memory

import (
	"context"
	"testing"

	"github.com/fracturing-space/game/internal/service"
)

func TestCanonicalValidationHelpers(t *testing.T) {
	t.Parallel()

	if _, err := validateCampaignID("   "); err == nil {
		t.Fatal("validateCampaignID(blank) error = nil, want failure")
	}
	if _, err := validateCampaignID(" camp-1 "); err == nil {
		t.Fatal("validateCampaignID(padded) error = nil, want failure")
	}
	if got, err := validateCampaignID("camp-1"); err != nil || got != "camp-1" {
		t.Fatalf("validateCampaignID(valid) = (%q,%v), want (camp-1,nil)", got, err)
	}

	if _, err := validateArtifactPath("   "); err == nil {
		t.Fatal("validateArtifactPath(blank) error = nil, want failure")
	}
	if _, err := validateArtifactPath(" notes.md "); err == nil {
		t.Fatal("validateArtifactPath(padded) error = nil, want failure")
	}
	if _, err := validateArtifactPath("/notes.md"); err == nil {
		t.Fatal("validateArtifactPath(leading slash) error = nil, want failure")
	}
	if got, err := validateArtifactPath("notes.md"); err != nil || got != "notes.md" {
		t.Fatalf("validateArtifactPath(valid) = (%q,%v), want (notes.md,nil)", got, err)
	}
}

func TestStoresRejectPaddedBoundaryInputs(t *testing.T) {
	t.Parallel()

	artifactStore := NewArtifactStore()
	if _, _, err := artifactStore.GetArtifact(context.Background(), " camp-1 ", "notes.md"); err == nil {
		t.Fatal("GetArtifact(padded campaign) error = nil, want failure")
	}
	if _, _, err := artifactStore.GetArtifact(context.Background(), "camp-1", " notes.md "); err == nil {
		t.Fatal("GetArtifact(padded path) error = nil, want failure")
	}
	if _, err := artifactStore.ListArtifacts(context.Background(), " camp-1 "); err == nil {
		t.Fatal("ListArtifacts(padded campaign) error = nil, want failure")
	}
	if err := artifactStore.PutArtifact(context.Background(), service.Artifact{CampaignID: " camp-1 ", Path: "notes.md"}); err == nil {
		t.Fatal("PutArtifact(padded campaign) error = nil, want failure")
	}

	projectionStore := NewProjectionStore()
	if _, _, err := projectionStore.GetProjection(context.Background(), " camp-1 "); err == nil {
		t.Fatal("GetProjection(padded campaign) error = nil, want failure")
	}
	if err := projectionStore.SaveProjection(context.Background(), service.ProjectionSnapshot{CampaignID: " camp-1 "}); err == nil {
		t.Fatal("SaveProjection(padded campaign) error = nil, want failure")
	}
	if _, _, err := projectionStore.GetWatermark(context.Background(), " camp-1 "); err == nil {
		t.Fatal("GetWatermark(padded campaign) error = nil, want failure")
	}
	if err := projectionStore.SaveWatermark(context.Background(), service.ProjectionWatermark{CampaignID: " camp-1 "}); err == nil {
		t.Fatal("SaveWatermark(padded campaign) error = nil, want failure")
	}
	if _, err := projectionStore.ListCampaignsBySubject(context.Background(), " subject-1 ", 10); err == nil {
		t.Fatal("ListCampaignsBySubject(padded subject) error = nil, want failure")
	}
}
