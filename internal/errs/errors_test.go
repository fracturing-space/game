package errs

import (
	"errors"
	"testing"
)

func TestErrorError(t *testing.T) {
	t.Parallel()

	var nilErr *Error
	if got := nilErr.Error(); got != "" {
		t.Fatalf("(*Error)(nil).Error() = %q, want empty", got)
	}

	err := &Error{Kind: KindInvalidArgument, Message: "boom"}
	if got, want := err.Error(), "boom"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestAsAndIs(t *testing.T) {
	t.Parallel()

	err := InvalidArgumentf("bad input")
	typed, ok := As(err)
	if !ok {
		t.Fatal("As() = false, want true")
	}
	if got, want := typed.Kind, KindInvalidArgument; got != want {
		t.Fatalf("kind = %q, want %q", got, want)
	}
	if !Is(err, KindInvalidArgument) {
		t.Fatal("Is(invalid argument) = false, want true")
	}
	if Is(err, KindNotFound) {
		t.Fatal("Is(not found) = true, want false")
	}

	wrapped := errors.New("boom")
	if _, ok := As(wrapped); ok {
		t.Fatal("As(non-domain error) = true, want false")
	}
}

func TestConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		kind Kind
		msg  string
	}{
		{name: "new", err: New(KindConflict, "conflict"), kind: KindConflict, msg: "conflict"},
		{name: "not found", err: NotFoundf("missing %s", "campaign"), kind: KindNotFound, msg: "missing campaign"},
		{name: "already exists", err: AlreadyExistsf("duplicate %s", "campaign"), kind: KindAlreadyExists, msg: "duplicate campaign"},
		{name: "conflict", err: Conflictf("conflict %s", "campaign"), kind: KindConflict, msg: "conflict campaign"},
		{name: "failed precondition", err: FailedPreconditionf("blocked %s", "campaign"), kind: KindFailedPrecondition, msg: "blocked campaign"},
		{name: "invalid argument", err: InvalidArgumentf("invalid %s", "campaign"), kind: KindInvalidArgument, msg: "invalid campaign"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			typed, ok := As(tt.err)
			if !ok {
				t.Fatal("As() = false, want true")
			}
			if got, want := typed.Kind, tt.kind; got != want {
				t.Fatalf("kind = %q, want %q", got, want)
			}
			if got, want := typed.Message, tt.msg; got != want {
				t.Fatalf("message = %q, want %q", got, want)
			}
		})
	}
}
