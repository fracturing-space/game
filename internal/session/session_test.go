package session

import "testing"

func TestCommandAndEventTypes(t *testing.T) {
	t.Parallel()

	if got, want := (Start{}).CommandType(), CommandTypeStart; got != want {
		t.Fatalf("Start.CommandType() = %q, want %q", got, want)
	}
	if got, want := (End{}).CommandType(), CommandTypeEnd; got != want {
		t.Fatalf("End.CommandType() = %q, want %q", got, want)
	}
	if got, want := (Started{}).EventType(), EventTypeStarted; got != want {
		t.Fatalf("Started.EventType() = %q, want %q", got, want)
	}
	if got, want := (Ended{}).EventType(), EventTypeEnded; got != want {
		t.Fatalf("Ended.EventType() = %q, want %q", got, want)
	}
}

func TestStatusValid(t *testing.T) {
	t.Parallel()

	if !StatusActive.Valid() {
		t.Fatal("StatusActive.Valid() = false, want true")
	}
	if !StatusEnded.Valid() {
		t.Fatal("StatusEnded.Valid() = false, want true")
	}
	if Status("BROKEN").Valid() {
		t.Fatal("Status(\"BROKEN\").Valid() = true, want false")
	}
}

func TestCloneHelpers(t *testing.T) {
	t.Parallel()

	assignments := CloneAssignments([]CharacterControllerAssignment{
		{CharacterID: "char-2", ParticipantID: "part-2"},
		{CharacterID: "char-1", ParticipantID: "part-1"},
	})
	if got, want := assignments[0].CharacterID, "char-1"; got != want {
		t.Fatalf("first character id = %q, want %q", got, want)
	}

	record := CloneRecord(&Record{
		ID:     "sess-1",
		Name:   "Night Watch",
		Status: StatusActive,
		CharacterControllers: []CharacterControllerAssignment{
			{CharacterID: "char-2", ParticipantID: "part-2"},
			{CharacterID: "char-1", ParticipantID: "part-1"},
		},
	})
	if got, want := record.CharacterControllers[0].CharacterID, "char-1"; got != want {
		t.Fatalf("cloned character id = %q, want %q", got, want)
	}
	if got := CloneRecord(nil); got != nil {
		t.Fatalf("CloneRecord(nil) = %+v, want nil", got)
	}
}

func TestValidation(t *testing.T) {
	t.Parallel()

	if err := ValidateStart(Start{
		CharacterControllers: []CharacterControllerAssignment{{ParticipantID: "part-1"}},
	}); err == nil {
		t.Fatal("ValidateStart(missing character id) error = nil, want failure")
	}
	if err := ValidateStart(Start{
		CharacterControllers: []CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}); err != nil {
		t.Fatalf("ValidateStart(valid) error = %v", err)
	}

	if err := ValidateStarted(Started{Name: "Night Watch"}); err == nil {
		t.Fatal("ValidateStarted(missing session id) error = nil, want failure")
	}
	if err := ValidateStarted(Started{
		SessionID:            "sess-1",
		Name:                 "Night Watch",
		CharacterControllers: []CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}); err != nil {
		t.Fatalf("ValidateStarted(valid) error = %v", err)
	}

	if err := ValidateEnded(Ended{Name: "Night Watch"}); err == nil {
		t.Fatal("ValidateEnded(missing session id) error = nil, want failure")
	}
	if err := ValidateEnded(Ended{
		SessionID:            "sess-1",
		Name:                 "Night Watch",
		CharacterControllers: []CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: "part-1"}},
	}); err != nil {
		t.Fatalf("ValidateEnded(valid) error = %v", err)
	}
}

func TestSpecsNormalizeMessages(t *testing.T) {
	t.Parallel()

	normalizedStart, err := StartCommandSpec.NormalizeMessage(Start{
		Name: " Night Watch ",
		CharacterControllers: []CharacterControllerAssignment{
			{CharacterID: "char-2", ParticipantID: "part-2"},
			{CharacterID: "char-1", ParticipantID: "part-1"},
		},
	})
	if err != nil {
		t.Fatalf("StartCommandSpec.NormalizeMessage() error = %v", err)
	}
	startMessage := normalizedStart.(Start)
	if got, want := startMessage.Name, "Night Watch"; got != want {
		t.Fatalf("normalized start name = %q, want %q", got, want)
	}
	if got, want := startMessage.CharacterControllers[0].CharacterID, "char-1"; got != want {
		t.Fatalf("normalized start character id = %q, want %q", got, want)
	}

	normalizedStarted, err := StartedEventSpec.NormalizeMessage(Started{
		SessionID: "sess-1",
		Name:      " Night Watch ",
		CharacterControllers: []CharacterControllerAssignment{
			{CharacterID: "char-2", ParticipantID: "part-2"},
			{CharacterID: "char-1", ParticipantID: "part-1"},
		},
	})
	if err != nil {
		t.Fatalf("StartedEventSpec.NormalizeMessage() error = %v", err)
	}
	startedMessage := normalizedStarted.(Started)
	if got, want := startedMessage.Name, "Night Watch"; got != want {
		t.Fatalf("normalized started name = %q, want %q", got, want)
	}
	if got, want := startedMessage.CharacterControllers[0].CharacterID, "char-1"; got != want {
		t.Fatalf("normalized started character id = %q, want %q", got, want)
	}

	normalizedEnded := normalizeEnded(Ended{
		SessionID: "sess-1",
		Name:      " Night Watch ",
		CharacterControllers: []CharacterControllerAssignment{
			{CharacterID: "char-2", ParticipantID: "part-2"},
			{CharacterID: "char-1", ParticipantID: "part-1"},
		},
	})
	if got, want := normalizedEnded.Name, "Night Watch"; got != want {
		t.Fatalf("normalizeEnded() name = %q, want %q", got, want)
	}
	if got, want := normalizedEnded.CharacterControllers[0].CharacterID, "char-1"; got != want {
		t.Fatalf("normalizeEnded() first character id = %q, want %q", got, want)
	}
}

func TestValidateAssignmentsBranches(t *testing.T) {
	t.Parallel()

	if err := validateAssignments([]CharacterControllerAssignment{{
		CharacterID:   "char-1",
		ParticipantID: "part-1",
	}, {
		CharacterID:   "char-1",
		ParticipantID: "part-2",
	}}); err == nil {
		t.Fatal("validateAssignments(duplicate character) error = nil, want failure")
	}
	if err := validateAssignments([]CharacterControllerAssignment{{
		CharacterID:   "char-1",
		ParticipantID: "",
	}}); err == nil {
		t.Fatal("validateAssignments(blank participant) error = nil, want failure")
	}
}
