package versions

import (
	"errors"
	"fmt"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

var (
	// ErrInvalidActualAnswersState is returned when ActualAnswers is already
	// greater than ExpectedAnswers.
	ErrInvalidActualAnswersState = errors.New("invalid actual answers state")

	// ErrProhibitedRequestID is returned when [RequestsV1.RegisterIncomingRequest]
	// is called with [PROHIBITED_REQUEST_ID].
	ErrProhibitedRequestID = errors.New("prohibited request id")

	// ErrRequestNotFound is returned when [RequestsV1.RegisterResponse] names an
	// ID that is not registered.
	ErrRequestNotFound = errors.New("request is not found")

	// ErrNotResponse is returned when [RequestsV1.RegisterResponse] is called for
	// an incoming request row.
	ErrNotResponse = errors.New("it's not response")

	// ErrNoFreeRequestID is returned when every non-zero request ID is already
	// registered.
	ErrNoFreeRequestID = errors.New("no free request ID")

	// ErrVersionNeitherGrantedNorActivated is returned when a register or
	// response method is called before the version is both granted and active.
	ErrVersionNeitherGrantedNorActivated = errors.New("version neither granted nor activated")
)

// ErrorRequestIDUsed is returned when [RequestsV1.RegisterIncomingRequest]
// names an ID that is already registered.
type ErrorRequestIDUsed struct {
	id fields.RequestID
}

// NewErrorRequestIDUsed returns an ErrorRequestIDUsed for id.
func NewErrorRequestIDUsed(id fields.RequestID) ErrorRequestIDUsed {
	return ErrorRequestIDUsed{id: id}
}

// ID returns the request ID that is already in use.
func (e ErrorRequestIDUsed) ID() fields.RequestID {
	return e.id
}

// Error returns a human-readable summary of the error.
func (e ErrorRequestIDUsed) Error() string {
	return fmt.Sprintf("id %v is used", e.id)
}

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
