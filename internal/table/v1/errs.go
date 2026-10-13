package table_v1

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidActualAnswersState is returned when ActualAnswers is already
	// greater than ExpectedAnswers.
	ErrInvalidActualAnswersState = errors.New("invalid actual answers state")

	// ErrRequestNotFound is returned when [RequestsV1.RegisterResponse] names an
	// ID that is not registered.
	ErrRequestNotFound = errors.New("request is not found")

	// ErrNoFreeRequestID is returned when every non-zero request ID is already
	// registered.
	ErrNoFreeRequestID = errors.New("no free request ID")

	// ErrVersionNeitherGrantedNorActivated is returned when a register or
	// response method is called before the version is both granted and active.
	ErrVersionNeitherGrantedNorActivated = errors.New("version neither granted nor activated")

	// ErrNilResponseV1Chan is returned when [RequestsV1.RegisterRequest] is called
	// with a nil [ResponseV1Chan].
	ErrNilResponseV1Chan = errors.New("expected ResponseV1Chan, got nil")
)

// ErrorUnexpectedAnswers is returned when a response would push ActualAnswers
// past ExpectedAnswers.
type ErrorUnexpectedAnswers struct {
	expected uint32
	got      uint32
}

// NewErrorUnexpectedAnswers returns an ErrorUnexpectedAnswers for the remaining
// expected count and the number of answers received.
func NewErrorUnexpectedAnswers(expected, got uint32) ErrorUnexpectedAnswers {
	return ErrorUnexpectedAnswers{expected: expected, got: got}
}

// Expected returns how many answers were still allowed.
func (e ErrorUnexpectedAnswers) Expected() uint32 {
	return e.expected
}

// Got returns how many answers were offered.
func (e ErrorUnexpectedAnswers) Got() uint32 {
	return e.got
}

// Error returns a human-readable summary of the error.
func (e ErrorUnexpectedAnswers) Error() string {
	return fmt.Sprintf("expected %v answers, but got %v", e.expected, e.got)
}
