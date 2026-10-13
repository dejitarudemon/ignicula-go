package runners

import (
	"errors"
	"fmt"
)

var (
	// ErrNilRequestsChan is returned by NewRunner when the requests channel is nil.
	ErrNilRequestsChan = errors.New("expected requests chan, got nil")
	// ErrNilRequestsV1 is returned by NewRunner when the requests table is nil.
	ErrNilRequestsV1 = errors.New("expected table_v1.RequestsV1, got nil")
	// ErrWaitReady is returned by WaitReady when the ready channel yields an unexpected value.
	ErrWaitReady = errors.New("wait ready error")
	// ErrActivating is returned when an operation cannot run because activation is in progress.
	ErrActivating = errors.New("runner is activating")
	// ErrPingExhausted is returned when Ping used up all retry attempts.
	ErrPingExhausted = errors.New("ping exhausted attempts")
	// ErrFailedCastBatchAnswer is returned when an Answer cannot be cast to BatchAnswer.
	ErrFailedCastBatchAnswer = errors.New("failed to cast Answer to BatchAnswer")
	// ErrRunnerClosed is returned when the runner has been closed.
	ErrRunnerClosed = errors.New("runner closed")
)

type CloseConnection struct {
	err error
}

func NewCloseConnection(err error) CloseConnection {
	return CloseConnection{err: err}
}

func (e CloseConnection) Error() string {
	return fmt.Sprintf("unprocessiable error: %v", e.err)
}

func (e CloseConnection) Unwrap() error {
	return e.err
}

type LogAndIgnore struct {
	err error
}

func NewLogAndIgnore(err error) LogAndIgnore {
	return LogAndIgnore{err: err}
}

func (e LogAndIgnore) Error() string {
	return e.err.Error()
}

func (e LogAndIgnore) Unwrap() error {
	return e.err
}

type ProtocolViolationByServer struct {
	msg string
}

func NewProtocolViolationByServer(msg string) ProtocolViolationByServer {
	return ProtocolViolationByServer{msg: msg}
}

func (e ProtocolViolationByServer) Error() string {
	return fmt.Sprintf("Protocol Violation By Server: %v", e.msg)
}

// ServerError is a protocol-valid failure reported by the server (for example an
// ErrorAnswer). It is not a protocol violation: the peer followed the wire rules.
type ServerError struct {
	err error
}

// NewServerError wraps err as a [ServerError].
func NewServerError(err error) ServerError {
	return ServerError{err: err}
}

// Error returns a human-readable summary of the error.
func (e ServerError) Error() string {
	return fmt.Sprintf("server error: %v", e.err)
}

// Unwrap returns the underlying server-reported error.
func (e ServerError) Unwrap() error {
	return e.err
}

type InternalError struct {
	err error
}

func NewInternalError(err error) InternalError {
	return InternalError{err: err}
}

func (e InternalError) Error() string {
	return fmt.Sprintf("Internal Error: %v", e.err)
}

func (e InternalError) Unwrap() error {
	return e.err
}
