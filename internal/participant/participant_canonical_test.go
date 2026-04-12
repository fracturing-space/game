package participant

import "testing"

func TestCanonicalValidationBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "update padded id", err: ValidateUpdate(Update{ParticipantID: " part-1 ", Name: "Zoe", Access: AccessMember})},
		{name: "bind padded id", err: ValidateBind(Bind{ParticipantID: " part-1 ", SubjectID: "subject-1"})},
		{name: "bind padded subject", err: ValidateBind(Bind{ParticipantID: "part-1", SubjectID: " subject-1 "})},
		{name: "unbind padded id", err: ValidateUnbind(Unbind{ParticipantID: " part-1 "})},
		{name: "leave padded id", err: ValidateLeave(Leave{ParticipantID: " part-1 "})},
		{name: "joined padded id", err: ValidateJoined(Joined{ParticipantID: " part-1 ", Name: "Zoe", Access: AccessMember})},
		{name: "identity padded access", err: validateIdentity(Access(" MEMBER "), "")},
		{name: "identity padded subject", err: validateIdentity(AccessMember, " subject-1 ")},
	}
	for _, test := range tests {
		if test.err == nil {
			t.Fatalf("%s error = nil, want failure", test.name)
		}
	}
}
