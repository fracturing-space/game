package participant

import "testing"

func TestParticipantLifecycleValidationAndNormalization(t *testing.T) {
	t.Parallel()

	if err := validateIdentity(Access("BROKEN"), ""); err == nil {
		t.Fatal("validateIdentity(invalid access) error = nil, want failure")
	}

	if err := ValidateUpdate(Update{}); err == nil {
		t.Fatal("ValidateUpdate() error = nil, want failure")
	}
	if err := ValidateUpdate(Update{ParticipantID: "part-1", Name: "zoe", Access: AccessMember}); err != nil {
		t.Fatalf("ValidateUpdate(valid) error = %v", err)
	}

	normalized := normalizeUpdate(Update{ParticipantID: "part-1", Name: " zoe ", Access: AccessMember})
	if got, want := normalized.Name, "zoe"; got != want {
		t.Fatalf("normalized update name = %q, want %q", got, want)
	}

	if err := ValidateBind(Bind{ParticipantID: "part-1", SubjectID: "subject-1"}); err != nil {
		t.Fatalf("ValidateBind(valid) error = %v", err)
	}
	if err := ValidateUnbind(Unbind{ParticipantID: "part-1"}); err != nil {
		t.Fatalf("ValidateUnbind(valid) error = %v", err)
	}
	if err := ValidateLeave(Leave{ParticipantID: "part-1"}); err != nil {
		t.Fatalf("ValidateLeave(valid) error = %v", err)
	}
	if err := ValidateUpdated(Updated{ParticipantID: "part-1", Name: "zoe", Access: AccessMember}); err != nil {
		t.Fatalf("ValidateUpdated(valid) error = %v", err)
	}
	if err := ValidateBound(Bound{ParticipantID: "part-1", SubjectID: "subject-1"}); err != nil {
		t.Fatalf("ValidateBound(valid) error = %v", err)
	}
	if err := ValidateUnbound(Unbound{ParticipantID: "part-1"}); err != nil {
		t.Fatalf("ValidateUnbound(valid) error = %v", err)
	}
	if err := ValidateLeft(Left{ParticipantID: "part-1"}); err != nil {
		t.Fatalf("ValidateLeft(valid) error = %v", err)
	}

	if got := normalizeBind(Bind{ParticipantID: "part-1", SubjectID: "subject-1"}); got.ParticipantID != "part-1" || got.SubjectID != "subject-1" {
		t.Fatalf("normalizeBind() = %+v, want passthrough", got)
	}
	if got := normalizeUnbind(Unbind{ParticipantID: "part-1"}); got.ParticipantID != "part-1" {
		t.Fatalf("normalizeUnbind() = %+v, want passthrough", got)
	}
	if got := normalizeLeave(Leave{ParticipantID: "part-1", Reason: " left "}); got.Reason != "left" {
		t.Fatalf("normalizeLeave() = %+v, want trimmed reason", got)
	}
	if got := normalizeJoined(Joined{ParticipantID: "part-1", Name: " zoe "}); got.Name != "zoe" {
		t.Fatalf("normalizeJoined() = %+v, want trimmed name", got)
	}
	if got := normalizeUpdated(Updated{ParticipantID: "part-1", Name: " zoe "}); got.Name != "zoe" {
		t.Fatalf("normalizeUpdated() = %+v, want trimmed name", got)
	}
	if got := normalizeBound(Bound{ParticipantID: "part-1", SubjectID: "subject-1"}); got.SubjectID != "subject-1" {
		t.Fatalf("normalizeBound() = %+v, want passthrough", got)
	}
	if got := normalizeUnbound(Unbound{ParticipantID: "part-1"}); got.ParticipantID != "part-1" {
		t.Fatalf("normalizeUnbound() = %+v, want passthrough", got)
	}
	if got := normalizeLeft(Left{ParticipantID: "part-1"}); got.ParticipantID != "part-1" {
		t.Fatalf("normalizeLeft() = %+v, want passthrough", got)
	}
}
