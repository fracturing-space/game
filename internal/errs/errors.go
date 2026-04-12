package errs

import (
	"errors"
	"fmt"
)

// Kind identifies one transport-relevant domain failure class.
type Kind string

const (
	KindNotFound           Kind = "not_found"
	KindAlreadyExists      Kind = "already_exists"
	KindConflict           Kind = "conflict"
	KindFailedPrecondition Kind = "failed_precondition"
	KindInvalidArgument    Kind = "invalid_argument"
)

// Error reports one classified domain failure.
type Error struct {
	Kind    Kind
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Is reports whether err is a classified domain error of the supplied kind.
func Is(err error, kind Kind) bool {
	domainErr, ok := As(err)
	return ok && domainErr.Kind == kind
}

// As returns the typed domain error when present.
func As(err error) (*Error, bool) {
	var domainErr *Error
	ok := errors.As(err, &domainErr)
	return domainErr, ok
}

// New constructs one classified domain error.
func New(kind Kind, message string) error {
	return &Error{Kind: kind, Message: message}
}

// NotFoundf constructs one not-found domain error.
func NotFoundf(format string, args ...any) error {
	return New(KindNotFound, fmt.Sprintf(format, args...))
}

// AlreadyExistsf constructs one already-exists domain error.
func AlreadyExistsf(format string, args ...any) error {
	return New(KindAlreadyExists, fmt.Sprintf(format, args...))
}

// Conflictf constructs one conflict domain error.
func Conflictf(format string, args ...any) error {
	return New(KindConflict, fmt.Sprintf(format, args...))
}

// FailedPreconditionf constructs one failed-precondition domain error.
func FailedPreconditionf(format string, args ...any) error {
	return New(KindFailedPrecondition, fmt.Sprintf(format, args...))
}

// InvalidArgumentf constructs one invalid-argument domain error.
func InvalidArgumentf(format string, args ...any) error {
	return New(KindInvalidArgument, fmt.Sprintf(format, args...))
}
