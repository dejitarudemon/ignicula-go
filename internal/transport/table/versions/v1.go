package versions

import (
	"sync"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

// PROHIBITED_REQUEST_ID is the Handshake request ID on the wire.
// It is never allocated for an outgoing request and is rejected by
// [RequestsV1.RegisterIncomingRequest]. Handshake itself is not registered in
// this table: registration starts only after a successful Handshake.
const (
	PROHIBITED_REQUEST_ID = 0
)

// RequestV1 is the state of one registered request ID.
//
// IsOutcoming is true for requests this side sent and still expects answers for.
// ExpectedAnswers is how many answers complete the request; ActualAnswers is how
// many have been recorded. Incoming rows from [RequestsV1.RegisterIncomingRequest]
// are stored with ExpectedAnswers and ActualAnswers both set to 1 (a client in v1
// only needs the ID reserved, typically for a server Ping).
type RequestV1 struct {
	IsOutcoming bool

	ExpectedAnswers uint32
	ActualAnswers   uint32
}

// NewRequestV1 returns an outgoing or incoming request that expects at least one
// answer. A zero expected value is raised to 1. ActualAnswers starts at 0.
func NewRequestV1(IsOutcoming bool, expected uint32) RequestV1 {
	return RequestV1{
		IsOutcoming:     IsOutcoming,
		ExpectedAnswers: max(1, expected),
	}
}

// NewRequestV1WithActual is like [NewRequestV1], but sets ActualAnswers to actual.
// A zero expected value is raised to 1. actual is stored as given, including
// values greater than ExpectedAnswers.
func NewRequestV1WithActual(IsOutcoming bool, expected uint32, actual uint32) RequestV1 {
	return RequestV1{
		IsOutcoming:     IsOutcoming,
		ExpectedAnswers: max(1, expected),
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
// Outgoing IDs are allocated by [RequestsV1.RegisterOutcomingRequest] from a
// wrapping cursor that skips [PROHIBITED_REQUEST_ID]. Incoming IDs are stored
// as given by [RequestsV1.RegisterIncomingRequest]; zero is rejected.
// A completed outgoing row stays until [RequestsV1.Terminate] so the ID remains
// unique while the caller still holds it.
//
// Handshake is not registered: until an official Handshake succeeds, callers
// must not treat the table as ready to register requests. After success,
// [RequestsV1.Grant] and [RequestsV1.Activate] record that the version is
// granted and active. [RequestsV1.RegisterOutcomingRequest],
// [RequestsV1.RegisterIncomingRequest], and [RequestsV1.RegisterResponse]
// require both flags; otherwise they return
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

// RegisterOutcomingRequest allocates a free non-zero request ID, stores an
// outgoing [RequestV1] that expects at least expected answers, and returns
// that ID.
//
// Both granted and active must be set, or the error is
// [ErrVersionNeitherGrantedNorActivated]. A zero expected value is raised to 1.
// If every non-zero ID is already registered, id is 0 and the error is
// [ErrNoFreeRequestID].
func (r *RequestsV1) RegisterOutcomingRequest(expected uint32) (id fields.RequestID, err error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if !r.active || !r.granted {
		return 0, ErrVersionNeitherGrantedNorActivated
	}

	if err = r.increase(); err != nil {
		return
	}

	id = r.cursor
	r.requests[id] = NewRequestV1(true, expected)

	return
}

// RegisterIncomingRequest stores id as an incoming request that is already
// complete (ExpectedAnswers and ActualAnswers are both 1).
//
// Both granted and active must be set, or the error is
// [ErrVersionNeitherGrantedNorActivated]. [PROHIBITED_REQUEST_ID] returns
// [ErrProhibitedRequestID]. If id is already registered, the error is
// [ErrorRequestIDUsed]. On a v1 client this mainly reserves the ID against a
// server Ping; the row does not track further answers.
func (r *RequestsV1) RegisterIncomingRequest(id fields.RequestID) error {
	if id == PROHIBITED_REQUEST_ID {
		return ErrProhibitedRequestID
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	if !r.active || !r.granted {
		return ErrVersionNeitherGrantedNorActivated
	}

	if _, ok := r.requests[id]; ok {
		return NewErrorRequestIDUsed(id)
	}

	r.requests[id] = NewRequestV1WithActual(false, 1, 1)

	return nil
}

// RegisterResponse records got answers for the outgoing request id.
//
// Both granted and active must be set, or the error is
// [ErrVersionNeitherGrantedNorActivated]. waitNext is true when more answers
// are still expected. Missing id returns [ErrRequestNotFound]. An incoming row
// returns [ErrNotResponse]. An overflow returns [ErrorUnexpectedAnswers]; an
// already-invalid row returns [ErrInvalidActualAnswersState]. A successful call
// that completes the expected total leaves the row in the map until
// [RequestsV1.Terminate].
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

	if !request.IsOutcoming {
		return false, ErrNotResponse
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
// The returned value is a copy; changing it does not change the table.
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
