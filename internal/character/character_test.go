package character

import "testing"

func TestCommandAndEventTypes(t *testing.T) {
	t.Parallel()

	if got, want := (Create{}).CommandType(), CommandTypeCreate; got != want {
		t.Fatalf("Create.CommandType() = %q, want %q", got, want)
	}
	if got, want := (Created{}).EventType(), EventTypeCreated; got != want {
		t.Fatalf("Created.EventType() = %q, want %q", got, want)
	}
}

func TestValidateCreateAndCreated(t *testing.T) {
	t.Parallel()

	normalized, err := CreateCommandSpec.NormalizeMessage(Create{ParticipantID: "part-1", Name: " luna "})
	if err != nil {
		t.Fatalf("CreateCommandSpec.NormalizeMessage() error = %v", err)
	}
	createMessage := normalized.(Create)
	if got, want := createMessage.Name, "luna"; got != want {
		t.Fatalf("normalized create name = %q, want %q", got, want)
	}

	if err := ValidateCreate(Create{ParticipantID: "part-1", Name: "luna"}); err != nil {
		t.Fatalf("ValidateCreate(valid) error = %v", err)
	}
	if err := ValidateCreated(Created{CharacterID: "char-1", ParticipantID: "part-1", Name: "luna"}); err != nil {
		t.Fatalf("ValidateCreated(valid) error = %v", err)
	}

	normalizedCreated, err := CreatedEventSpec.NormalizeMessage(Created{
		CharacterID:   "char-1",
		ParticipantID: "part-1",
		Name:          " luna ",
	})
	if err != nil {
		t.Fatalf("CreatedEventSpec.NormalizeMessage() error = %v", err)
	}
	createdMessage := normalizedCreated.(Created)
	if got, want := createdMessage.Name, "luna"; got != want {
		t.Fatalf("normalized created name = %q, want %q", got, want)
	}
}
