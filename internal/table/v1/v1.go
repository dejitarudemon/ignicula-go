package table_v1

import (
	"sync"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/frame"
)

// PROHIBITED_REQUEST_ID is the Handshake request ID on the wire.
// It is never allocated by [RequestsV1.RegisterRequest]. Handshake itself is
// not registered in this table: registration starts only after a successful
// Handshake.
const (
	PROHIBITED_REQUEST_ID = 0
)

// ResponseV1 is one answer delivered on a [ResponseV1Chan].
//
// Err is set when delivery carries a failure instead of a normal frame.
// Closing the channel signals that no further answers will follow for the
// request; intermediate answers leave the channel open.
type ResponseV1 struct {
	Response frame.Frame
	Err      error
}

// ResponseV1Chan is the bidirectional caller-facing channel stored on a
// [RequestV1]. The caller receives from it; the consumer sends and closes it.
//
// The registering caller must not close it. The consumer (for example the v1
// runner) closes the channel when the request is finished or failed; that close
// is the end-of-stream signal for the caller.
type ResponseV1Chan chan ResponseV1

// RequestV1 is the state of one registered request ID.
//
// Command is the request command the answers must respond to.
// ExpectedAnswers is how many answers complete the request; ActualAnswers is
// how many have been recorded. ToCaller receives [ResponseV1] values for the
// waiting caller; the channel value is shared even when [RequestsV1.Find]
// returns a copy of the row.
type RequestV1 struct {
	Command fields.Command

	ExpectedAnswers uint32
	ActualAnswers   uint32

	ToCaller ResponseV1Chan
}

// NewRequestV1 returns a request that expects at least one answer.
// A zero expected value is raised to 1. ActualAnswers starts at 0.
func NewRequestV1(expected uint32, command fields.Command, toCaller ResponseV1Chan) RequestV1 {
	return RequestV1{
		ExpectedAnswers: max(1, expected),
		Command:         command,
		ToCaller:        toCaller,
	}
}

// NewRequestV1WithActual is like [NewRequestV1], but sets ActualAnswers to actual.
// A zero expected value is raised to 1. actual is stored as given, including
// values greater than ExpectedAnswers.
func NewRequestV1WithActual(expected uint32, command fields.Command, toCaller ResponseV1Chan, actual uint32) RequestV1 {
	return RequestV1{
		ExpectedAnswers: max(1, expected),
		Command:         command,
		ToCaller:        toCaller,
		ActualAnswers:   actual,
	}
}

// add records answers toward ExpectedAnswers.
//
// full is true when ActualAnswers equals ExpectedAnswers after the update.
// It returns an error when ActualAnswers is already greater than ExpectedAnswers,
// or when answers would push ActualAnswers past ExpectedAnswers. A zero answers
// value is accepted and leaves the counters unchanged.
func (r *RequestV1) add(answers uint32) (full bool, err error) {
	if r.ActualAnswers > r.ExpectedAnswers {
		return false, ErrInvalidActualAnswersState
	}
	if answers > r.ExpectedAnswers-r.ActualAnswers {
		return false, NewErrorUnexpectedAnswers(r.ExpectedAnswers-r.ActualAnswers, answers)
	}

	r.ActualAnswers += answers
	return r.ActualAnswers == r.ExpectedAnswers, nil
}

// RequestsV1 tracks active request IDs and grant/active flags for protocol v1
// on one connection.
//
// IDs are allocated by [RequestsV1.RegisterRequest] from a wrapping cursor that
// skips [PROHIBITED_REQUEST_ID]. A completed row stays until
// [RequestsV1.Terminate] so the ID remains unique while the caller still holds it.
//
// Handshake is not registered: until an official Handshake succeeds, callers
// must not treat the table as ready to register requests. After success,
// [RequestsV1.Grant] and [RequestsV1.Activate] record that the version is
// granted and active. [RequestsV1.RegisterRequest] and
// [RequestsV1.RegisterResponse] require both flags; otherwise they return
// [ErrVersionNeitherGrantedNorActivated]. The flags do not touch the request map.
//
// Methods are safe for concurrent use. A RequestsV1 contains a mutex and must
// not be copied; share the pointer returned by [NewRequestsV1]. The zero
// RequestsV1 must not be used.
type RequestsV1 struct {
	requests map[fields.RequestID]RequestV1

	granted bool
	active  bool

	cursor fields.RequestID

	mx sync.RWMutex
}

// NewRequestsV1 returns an empty request set with grant and active cleared.
func NewRequestsV1() *RequestsV1 {
	return &RequestsV1{
		requests: make(map[fields.RequestID]RequestV1),
	}
}

// Grant records whether protocol v1 is granted on the connection after a
// successful Handshake. It does not change the request map or the active flag.
// Until granted, the table is not considered ready to register ordinary requests.
func (r *RequestsV1) Grant(b bool) {
	r.mx.Lock()
	defer r.mx.Unlock()

	r.granted = b
}

// Activate records whether protocol v1 is active on the connection after a
// successful Handshake. It does not change the request map or the granted flag.
func (r *RequestsV1) Activate(b bool) {
	r.mx.Lock()
	defer r.mx.Unlock()

	r.active = b
}

// RegisterRequest allocates a free non-zero request ID, stores a [RequestV1]
// for command with toCaller that expects at least expected answers, and
// returns that ID.
//
// toCaller must be non-nil, or the error is [ErrNilResponseV1Chan]. The caller
// must not close toCaller; the consumer closes it when the request ends.
// Both granted and active must be set, or the error is
// [ErrVersionNeitherGrantedNorActivated]. A zero expected value is raised to 1.
// If every non-zero ID is already registered, id is 0 and the error is
// [ErrNoFreeRequestID].
func (r *RequestsV1) RegisterRequest(command fields.Command, toCaller ResponseV1Chan, expected uint32) (id fields.RequestID, err error) {
	if toCaller == nil {
		return 0, ErrNilResponseV1Chan
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	if !r.active || !r.granted {
		return 0, ErrVersionNeitherGrantedNorActivated
	}

	if err = r.increase(); err != nil {
		return
	}

	id = r.cursor
	r.requests[id] = NewRequestV1(expected, command, toCaller)

	return
}

// RegisterResponse records got answers for the request id.
//
// Both granted and active must be set, or the error is
// [ErrVersionNeitherGrantedNorActivated]. waitNext is true when more answers
// are still expected; the consumer keeps [RequestV1.ToCaller] open in that case
// and closes it when waitNext is false (that close is the caller's end signal).
// Missing id returns [ErrRequestNotFound]. An overflow returns
// [ErrorUnexpectedAnswers]; an already-invalid row returns
// [ErrInvalidActualAnswersState]. A successful call that completes the expected
// total leaves the row in the map until [RequestsV1.Terminate].
func (r *RequestsV1) RegisterResponse(id fields.RequestID, got uint32) (waitNext bool, err error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if !r.active || !r.granted {
		return false, ErrVersionNeitherGrantedNorActivated
	}

	request, ok := r.requests[id]
	if !ok {
		return false, ErrRequestNotFound
	}

	full, err := request.add(got)
	if err != nil {
		return false, err
	}

	r.requests[id] = request
	waitNext = !full

	return
}

// Terminate removes id from the active set.
// An ID that is not registered is ignored.
func (r *RequestsV1) Terminate(id fields.RequestID) {
	r.mx.Lock()
	defer r.mx.Unlock()

	delete(r.requests, id)
}

// TerminateAll removes every registered request ID.
// Grant and active flags are left unchanged. An empty set is a no-op.
func (r *RequestsV1) TerminateAll() {
	r.mx.Lock()
	defer r.mx.Unlock()

	for id := range r.requests {
		delete(r.requests, id)
	}
}

// Find returns the [RequestV1] stored for id and whether it is registered.
// The returned value is a shallow copy; changing scalar fields does not change
// the table. [RequestV1.ToCaller] is shared with the stored row.
func (r *RequestsV1) Find(id fields.RequestID) (RequestV1, bool) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	request, ok := r.requests[id]
	return request, ok
}

// IsGranted reports whether [RequestsV1.Grant] last set the granted flag to true.
func (r *RequestsV1) IsGranted() bool {
	r.mx.RLock()
	defer r.mx.RUnlock()

	return r.granted
}

// IsActive reports whether [RequestsV1.Activate] last set the active flag to true.
func (r *RequestsV1) IsActive() bool {
	r.mx.RLock()
	defer r.mx.RUnlock()

	return r.active
}

// increase advances the cursor to the next free non-zero request ID.
// The caller must hold r.mx. It returns an error when no free ID remains.
func (r *RequestsV1) increase() error {
	next := r.cursor

	for {
		next++

		if next == r.cursor {
			return ErrNoFreeRequestID
		}

		if next == PROHIBITED_REQUEST_ID {
			continue
		}

		if _, exists := r.requests[next]; !exists {
			r.cursor = next
			return nil
		}
	}
}
