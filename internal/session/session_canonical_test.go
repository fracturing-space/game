package session

import "testing"

func TestCanonicalValidationBranches(t *testing.T) {
	t.Parallel()

	assignments := []CharacterControllerAssignment{{
		CharacterID:   "char-1",
		ParticipantID: "part-1",
	}}

	tests := []struct {
		name string
		err  error
	}{
		{name: "start padded character", err: ValidateStart(Start{Name: "Session", CharacterControllers: []CharacterControllerAssignment{{CharacterID: " char-1 ", ParticipantID: "part-1"}}})},
		{name: "start padded participant", err: ValidateStart(Start{Name: "Session", CharacterControllers: []CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: " part-1 "}}})},
		{name: "started padded id", err: ValidateStarted(Started{SessionID: " sess-1 ", Name: "Session", CharacterControllers: assignments})},
		{name: "ended padded id", err: ValidateEnded(Ended{SessionID: " sess-1 ", Name: "Session", CharacterControllers: assignments})},
		{name: "assignments padded character", err: validateAssignments([]CharacterControllerAssignment{{CharacterID: " char-1 ", ParticipantID: "part-1"}})},
		{name: "assignments padded participant", err: validateAssignments([]CharacterControllerAssignment{{CharacterID: "char-1", ParticipantID: " part-1 "}})},
	}
	for _, test := range tests {
		if test.err == nil {
			t.Fatalf("%s error = nil, want failure", test.name)
		}
	}
}
