package character

import "testing"

func TestCharacterLifecycleValidationAndNormalization(t *testing.T) {
	t.Parallel()

	if err := ValidateCreate(Create{Name: "Luna"}); err == nil {
		t.Fatal("ValidateCreate(missing participant id) error = nil, want failure")
	}

	if err := ValidateUpdate(Update{}); err == nil {
		t.Fatal("ValidateUpdate() error = nil, want failure")
	}
	if err := ValidateUpdate(Update{
		CharacterID:   "char-1",
		ParticipantID: "part-1",
		Name:          "Nova",
	}); err != nil {
		t.Fatalf("ValidateUpdate(valid) error = %v", err)
	}

	normalized := normalizeUpdate(Update{
		CharacterID:   "char-1",
		ParticipantID: "part-1",
		Name:          " nova ",
	})
	if got, want := normalized.Name, "nova"; got != want {
		t.Fatalf("normalized name = %q, want %q", got, want)
	}

	normalizedEvent := normalizeUpdated(Updated{
		CharacterID:   "char-1",
		ParticipantID: "part-1",
		Name:          " nova ",
	})
	if got, want := normalizedEvent.Name, "nova"; got != want {
		t.Fatalf("normalized event name = %q, want %q", got, want)
	}

	if err := ValidateDelete(Delete{}); err == nil {
		t.Fatal("ValidateDelete() error = nil, want failure")
	}
	if err := ValidateDeleted(Deleted{}); err == nil {
		t.Fatal("ValidateDeleted() error = nil, want failure")
	}
	if err := ValidateUpdated(Updated{
		CharacterID:   "char-1",
		ParticipantID: "part-1",
		Name:          "Nova",
	}); err != nil {
		t.Fatalf("ValidateUpdated(valid) error = %v", err)
	}
	if got := normalizeDelete(Delete{CharacterID: "char-1", Reason: " retired "}); got.Reason != "retired" {
		t.Fatalf("normalizeDelete() = %+v, want trimmed reason", got)
	}
	if got := normalizeDeleted(Deleted{CharacterID: "char-1"}); got.CharacterID != "char-1" {
		t.Fatalf("normalizeDeleted() = %+v, want passthrough", got)
	}
}
