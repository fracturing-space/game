package character

import "testing"

func TestValidateCanonicalIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "create padded owner", err: ValidateCreate(Create{ParticipantID: " part-1 ", Name: "Luna"})},
		{name: "update padded id", err: ValidateUpdate(Update{CharacterID: " char-1 ", ParticipantID: "part-1", Name: "Luna"})},
		{name: "delete padded id", err: ValidateDelete(Delete{CharacterID: " char-1 "})},
		{name: "created padded id", err: ValidateCreated(Created{CharacterID: " char-1 ", ParticipantID: "part-1", Name: "Luna"})},
	}
	for _, test := range tests {
		if test.err == nil {
			t.Fatalf("%s error = nil, want failure", test.name)
		}
	}
}
