package transfer

import (
	"errors"
	"fmt"
)

// ErrorKind classifies a business error. The HTTP layer maps each kind to a status code,
// so the domain never needs to know about HTTP.
type ErrorKind string

const (
	KindValidation ErrorKind = "validation"
	KindNotFound   ErrorKind = "not_found"
	KindConflict   ErrorKind = "conflict"
)

// Error is a business error with a human readable message.
type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func validationError(format string, args ...any) error {
	return &Error{Kind: KindValidation, Message: fmt.Sprintf(format, args...)}
}

func notFoundError(format string, args ...any) error {
	return &Error{Kind: KindNotFound, Message: fmt.Sprintf(format, args...)}
}

func conflictError(format string, args ...any) error {
	return &Error{Kind: KindConflict, Message: fmt.Sprintf(format, args...)}
}

// Errors a Repository returns so the service can react without knowing the database.
var (
	ErrRecordNotFound          = errors.New("record not found")
	ErrDuplicateIdempotencyKey = errors.New("idempotency key already used")
)
