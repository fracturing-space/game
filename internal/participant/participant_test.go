package participant

import "testing"

func TestCommandAndEventTypes(t *testing.T) {
	t.Parallel()

	if got, want := (Join{}).CommandType(), CommandTypeJoin; got != want {
		t.Fatalf("Join.CommandType() = %q, want %q", got, want)
	}
	if got, want := (Joined{}).EventType(), EventTypeJoined; got != want {
		t.Fatalf("Joined.EventType() = %q, want %q", got, want)
	}
}

func TestAccessValid(t *testing.T) {
	t.Parallel()

	if !AccessOwner.Valid() || !AccessMember.Valid() {
		t.Fatal("expected owner/member access to be valid")
	}
	if Access("").Valid() {
		t.Fatal("empty access should be invalid")
	}
}

func TestValidateJoinAndJoined(t *testing.T) {
	t.Parallel()

	normalized, err := JoinCommandSpec.NormalizeMessage(Join{Name: " louis "})
	if err != nil {
		t.Fatalf("JoinCommandSpec.NormalizeMessage() error = %v", err)
	}
	joinMessage := normalized.(Join)
	if got, want := joinMessage.Name, "louis"; got != want {
		t.Fatalf("normalized join name = %q, want %q", got, want)
	}
	if got, want := joinMessage.Access, AccessMember; got != want {
		t.Fatalf("normalized join access = %q, want %q", got, want)
	}

	if err := ValidateJoin(Join{Name: "louis", Access: AccessMember}); err != nil {
		t.Fatalf("ValidateJoin(member) error = %v", err)
	}
	if err := ValidateJoin(Join{Name: "louis", Access: AccessOwner, SubjectID: "subject-1"}); err != nil {
		t.Fatalf("ValidateJoin(owner) error = %v", err)
	}
	if err := ValidateJoined(Joined{ParticipantID: "part-1", Name: "louis", Access: AccessOwner, SubjectID: "subject-1"}); err != nil {
		t.Fatalf("ValidateJoined(valid) error = %v", err)
	}
	if err := ValidateJoined(Joined{ParticipantID: "part-1", Name: "louis", Access: AccessOwner}); err == nil {
		t.Fatal("ValidateJoined(owner missing subject) error = nil, want failure")
	}
}
