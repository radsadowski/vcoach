// Package errors defines domain error types. Services return these; the
// server layer maps them to HTTP responses. No business logic lives here.
package errors

import "fmt"

// ValidationError means the caller supplied invalid input.
type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string { return e.msg }

// NewValidationError builds a ValidationError with a formatted message.
func NewValidationError(format string, args ...any) *ValidationError {
	return &ValidationError{msg: fmt.Sprintf(format, args...)}
}

// NotFoundError means the requested entity does not exist (for this athlete).
type NotFoundError struct {
	msg string
}

func (e *NotFoundError) Error() string { return e.msg }

// NewNotFoundError builds a NotFoundError with a formatted message.
func NewNotFoundError(format string, args ...any) *NotFoundError {
	return &NotFoundError{msg: fmt.Sprintf(format, args...)}
}

// ConflictError means the request conflicts with existing state.
type ConflictError struct {
	msg string
}

func (e *ConflictError) Error() string { return e.msg }

// NewConflictError builds a ConflictError with a formatted message.
func NewConflictError(format string, args ...any) *ConflictError {
	return &ConflictError{msg: fmt.Sprintf(format, args...)}
}
