package memory

import (
	"context"
	"testing"

	"github.com/fracturing-space/game/internal/service"
)

func TestJournalAndProjectionStoreExtraBranches(t *testing.T) {
	t.Parallel()

	journal := NewJournal()
	if head, ok, err := journal.HeadSeq(context.Background(), "missing"); err != nil || ok || head != 0 {
		t.Fatalf("HeadSeq(missing) = (%d,%t,%v), want (0,false,nil)", head, ok, err)
	}

	store := NewProjectionStore()
	if err := store.SaveProjectionAndWatermark(context.Background(), service.ProjectionSnapshot{}, service.ProjectionWatermark{}); err == nil {
		t.Fatal("SaveProjectionAndWatermark(empty) error = nil, want failure")
	}
}
