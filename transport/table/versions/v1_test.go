package versions

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func TestNewRequestV1ClampsExpected(t *testing.T) {
	tests := []struct {
		name     string
		expected uint32
		want     uint32
	}{
		{"zero becomes one", 0, 1},
		{"one stays", 1, 1},
		{"large stays", math.MaxUint32, math.MaxUint32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRequestV1(true, tt.expected)
			if !got.IsOutcoming {
				t.Fatal("IsOutcoming = false, want true")
			}
			if got.ExpectedAnswers != tt.want {
				t.Fatalf("ExpectedAnswers = %d, want %d", got.ExpectedAnswers, tt.want)
			}
			if got.ActualAnswers != 0 {
				t.Fatalf("ActualAnswers = %d, want 0", got.ActualAnswers)
			}
		})
	}
}

func TestNewRequestV1WithActualKeepsOverflow(t *testing.T) {
	got := NewRequestV1WithActual(false, 0, 9)
	if got.IsOutcoming {
		t.Fatal("IsOutcoming = true, want false")
	}
	if got.ExpectedAnswers != 1 {
		t.Fatalf("ExpectedAnswers = %d, want 1", got.ExpectedAnswers)
	}
	if got.ActualAnswers != 9 {
		t.Fatalf("ActualAnswers = %d, want 9", got.ActualAnswers)
	}
}

func TestRequestV1Add(t *testing.T) {
	r := NewRequestV1(true, 3)

	full, err := r.add(1)
	if err != nil || full {
		t.Fatalf("add(1) = (%v, %v), want (false, nil)", full, err)
	}
	if r.ActualAnswers != 1 {
		t.Fatalf("ActualAnswers = %d, want 1", r.ActualAnswers)
	}

	full, err = r.add(0)
	if err != nil || full {
		t.Fatalf("add(0) = (%v, %v), want (false, nil)", full, err)
	}
	if r.ActualAnswers != 1 {
		t.Fatalf("ActualAnswers after zero = %d, want 1", r.ActualAnswers)
	}

	full, err = r.add(2)
	if err != nil || !full {
		t.Fatalf("add(2) = (%v, %v), want (true, nil)", full, err)
	}
	if r.ActualAnswers != 3 {
		t.Fatalf("ActualAnswers = %d, want 3", r.ActualAnswers)
	}
}

func TestRequestV1AddRejectsOverflow(t *testing.T) {
	r := NewRequestV1(true, 2)
	if _, err := r.add(1); err != nil {
		t.Fatalf("add(1) = %v", err)
	}

	_, err := r.add(2)
	got := asError[ErrorUnexpectedAnswers](t, err)
	if got.Expected() != 1 || got.Got() != 2 {
		t.Fatalf("error = %+v, want expected 1 got 2", got)
	}
	if r.ActualAnswers != 1 {
		t.Fatalf("ActualAnswers = %d, want unchanged 1", r.ActualAnswers)
	}
}

func TestRequestV1AddInvalidState(t *testing.T) {
	r := NewRequestV1WithActual(true, 1, 2)

	full, err := r.add(0)
	if full {
		t.Fatal("full = true, want false")
	}
	assertErrorIs(t, err, ErrInvalidActualAnswersState)
}

func TestRequestV1AddOnAlreadyFull(t *testing.T) {
	r := NewRequestV1WithActual(true, 2, 2)

	full, err := r.add(0)
	if err != nil || !full {
		t.Fatalf("add(0) on full = (%v, %v), want (true, nil)", full, err)
	}

	_, err = r.add(1)
	got := asError[ErrorUnexpectedAnswers](t, err)
	if got.Expected() != 0 || got.Got() != 1 {
		t.Fatalf("error = %+v, want expected 0 got 1", got)
	}
}

func TestNewRequestsV1Defaults(t *testing.T) {
	r := NewRequestsV1()

	if r.IsGranted() {
		t.Fatal("IsGranted() = true, want false")
	}
	if r.IsActive() {
		t.Fatal("IsActive() = true, want false")
	}
	if _, ok := r.Find(1); ok {
		t.Fatal("Find(1) = true, want false")
	}
}

func TestGrantAndActivateIndependent(t *testing.T) {
	r := NewRequestsV1()

	r.Grant(true)
	if !r.IsGranted() || r.IsActive() {
		t.Fatalf("after Grant(true): granted=%v active=%v", r.IsGranted(), r.IsActive())
	}

	r.Activate(true)
	if !r.IsGranted() || !r.IsActive() {
		t.Fatalf("after Activate(true): granted=%v active=%v", r.IsGranted(), r.IsActive())
	}

	r.Grant(false)
	if r.IsGranted() || !r.IsActive() {
		t.Fatalf("after Grant(false): granted=%v active=%v", r.IsGranted(), r.IsActive())
	}

	r.Activate(false)
	if r.IsGranted() || r.IsActive() {
		t.Fatalf("after Activate(false): granted=%v active=%v", r.IsGranted(), r.IsActive())
	}

	r.Grant(true)
	r.Activate(true)
	id, err := r.RegisterOutcomingRequest(1)
	if err != nil || id != 1 {
		t.Fatalf("RegisterOutcomingRequest = (%v, %v)", id, err)
	}
	r.Grant(false)
	r.Activate(false)
	r.Grant(true)
	r.Activate(true)
	if _, ok := r.Find(id); !ok {
		t.Fatal("Grant/Activate cleared the request map")
	}
}

func TestRegisterRequiresGrantedAndActive(t *testing.T) {
	r := NewRequestsV1()

	id, err := r.RegisterOutcomingRequest(1)
	if id != 0 {
		t.Fatalf("id = %v, want 0", id)
	}
	assertErrorIs(t, err, ErrVersionNeitherGrantedNorActivated)

	assertErrorIs(t, r.RegisterIncomingRequest(1), ErrVersionNeitherGrantedNorActivated)

	waitNext, err := r.RegisterResponse(1, 1)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	assertErrorIs(t, err, ErrVersionNeitherGrantedNorActivated)

	r.Grant(true)
	assertErrorIs(t, mustRegisterErr(t, r), ErrVersionNeitherGrantedNorActivated)

	r.Grant(false)
	r.Activate(true)
	assertErrorIs(t, mustRegisterErr(t, r), ErrVersionNeitherGrantedNorActivated)

	r.Grant(true)
	id, err = r.RegisterOutcomingRequest(1)
	if err != nil || id != 1 {
		t.Fatalf("with both flags = (%v, %v)", id, err)
	}
}

func mustRegisterErr(t *testing.T, r *RequestsV1) error {
	t.Helper()
	_, err := r.RegisterOutcomingRequest(1)
	return err
}

func readyRequestsV1(t *testing.T) *RequestsV1 {
	t.Helper()
	r := NewRequestsV1()
	r.Grant(true)
	r.Activate(true)
	return r
}

func TestRegisterOutcomingRequestAllocatesNonZero(t *testing.T) {
	r := readyRequestsV1(t)

	id, err := r.RegisterOutcomingRequest(0)
	if err != nil || id != 1 {
		t.Fatalf("RegisterOutcomingRequest(0) = (%v, %v), want (1, nil)", id, err)
	}

	req, ok := r.Find(id)
	if !ok {
		t.Fatal("Find(id) = false")
	}
	if !req.IsOutcoming || req.ExpectedAnswers != 1 || req.ActualAnswers != 0 {
		t.Fatalf("request = %+v, want outgoing expecting 1", req)
	}
	if _, ok := r.Find(PROHIBITED_REQUEST_ID); ok {
		t.Fatal("prohibited id 0 was allocated")
	}

	id, err = r.RegisterOutcomingRequest(5)
	if err != nil || id != 2 {
		t.Fatalf("RegisterOutcomingRequest(5) = (%v, %v), want (2, nil)", id, err)
	}

	req, ok = r.Find(id)
	if !ok {
		t.Fatal("Find(2) = false")
	}
	if req.ExpectedAnswers != 5 {
		t.Fatalf("ExpectedAnswers = %d, want 5", req.ExpectedAnswers)
	}
}

func TestRegisterOutcomingRequestSkipsOccupied(t *testing.T) {
	r := readyRequestsV1(t)

	if err := r.RegisterIncomingRequest(1); err != nil {
		t.Fatalf("RegisterIncomingRequest(1) = %v", err)
	}
	if err := r.RegisterIncomingRequest(3); err != nil {
		t.Fatalf("RegisterIncomingRequest(3) = %v", err)
	}

	id, err := r.RegisterOutcomingRequest(1)
	if err != nil || id != 2 {
		t.Fatalf("RegisterOutcomingRequest = (%v, %v), want (2, nil)", id, err)
	}

	id, err = r.RegisterOutcomingRequest(1)
	if err != nil || id != 4 {
		t.Fatalf("RegisterOutcomingRequest again = (%v, %v), want (4, nil)", id, err)
	}
}

func TestRegisterOutcomingRequestWraps(t *testing.T) {
	r := readyRequestsV1(t)
	r.cursor = math.MaxUint32

	id, err := r.RegisterOutcomingRequest(1)
	if err != nil || id != 1 {
		t.Fatalf("RegisterOutcomingRequest at max = (%v, %v), want (1, nil)", id, err)
	}
	if _, ok := r.Find(PROHIBITED_REQUEST_ID); ok {
		t.Fatal("wrap must skip prohibited id 0")
	}
}

func TestRegisterOutcomingRequestReusesAfterTerminate(t *testing.T) {
	r := readyRequestsV1(t)

	id, err := r.RegisterOutcomingRequest(1)
	if err != nil || id != 1 {
		t.Fatalf("first register = (%v, %v)", id, err)
	}
	r.Terminate(id)
	r.cursor = math.MaxUint32

	id, err = r.RegisterOutcomingRequest(1)
	if err != nil || id != 1 {
		t.Fatalf("register at max = (%v, %v), want reused 1", id, err)
	}
}

func TestIncreaseSkipsProhibitedAndBusy(t *testing.T) {
	r := NewRequestsV1()
	r.cursor = math.MaxUint32
	r.requests[1] = NewRequestV1(true, 1)

	if err := r.increase(); err != nil {
		t.Fatalf("increase = %v", err)
	}
	if r.cursor != 2 {
		t.Fatalf("cursor = %d, want 2 after skipping 0 and busy 1", r.cursor)
	}
}

func TestRegisterIncomingRequest(t *testing.T) {
	r := readyRequestsV1(t)

	if err := r.RegisterIncomingRequest(7); err != nil {
		t.Fatalf("RegisterIncomingRequest(7) = %v", err)
	}

	req, ok := r.Find(7)
	if !ok {
		t.Fatal("Find(7) = false")
	}
	if req.IsOutcoming || req.ExpectedAnswers != 1 || req.ActualAnswers != 1 {
		t.Fatalf("request = %+v, want completed incoming", req)
	}

	err := r.RegisterIncomingRequest(7)
	if got := asError[ErrorRequestIDUsed](t, err); got.ID() != 7 {
		t.Fatalf("ID() = %v, want 7", got.ID())
	}

	req, ok = r.Find(7)
	if !ok || req.IsOutcoming {
		t.Fatal("duplicate register mutated the stored row")
	}
}

func TestRegisterIncomingRequestRejectsZero(t *testing.T) {
	r := NewRequestsV1()

	err := r.RegisterIncomingRequest(PROHIBITED_REQUEST_ID)
	assertErrorIs(t, err, ErrProhibitedRequestID)
	if _, ok := r.Find(PROHIBITED_REQUEST_ID); ok {
		t.Fatal("prohibited id was stored")
	}
}

func TestRegisterIncomingConflictsWithOutgoing(t *testing.T) {
	r := readyRequestsV1(t)

	id, err := r.RegisterOutcomingRequest(1)
	if err != nil || id != 1 {
		t.Fatalf("RegisterOutcomingRequest = (%v, %v)", id, err)
	}

	err = r.RegisterIncomingRequest(id)
	if got := asError[ErrorRequestIDUsed](t, err); got.ID() != id {
		t.Fatalf("ID() = %v, want %v", got.ID(), id)
	}

	req, ok := r.Find(id)
	if !ok || !req.IsOutcoming {
		t.Fatal("conflict replaced the outgoing row")
	}
}

func TestRegisterResponse(t *testing.T) {
	r := readyRequestsV1(t)

	id, err := r.RegisterOutcomingRequest(3)
	if err != nil || id != 1 {
		t.Fatalf("RegisterOutcomingRequest = (%v, %v)", id, err)
	}

	waitNext, err := r.RegisterResponse(id, 1)
	if err != nil || !waitNext {
		t.Fatalf("RegisterResponse(1) = (%v, %v), want wait", waitNext, err)
	}

	req, _ := r.Find(id)
	if req.ActualAnswers != 1 {
		t.Fatalf("ActualAnswers = %d, want 1", req.ActualAnswers)
	}

	waitNext, err = r.RegisterResponse(id, 2)
	if err != nil || waitNext {
		t.Fatalf("RegisterResponse(2) = (%v, %v), want complete", waitNext, err)
	}

	req, ok := r.Find(id)
	if !ok {
		t.Fatal("completed request was removed without Terminate")
	}
	if req.ActualAnswers != 3 {
		t.Fatalf("ActualAnswers = %d, want 3", req.ActualAnswers)
	}
}

func TestRegisterResponseErrors(t *testing.T) {
	r := readyRequestsV1(t)

	waitNext, err := r.RegisterResponse(1, 1)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	assertErrorIs(t, err, ErrRequestNotFound)

	if err := r.RegisterIncomingRequest(2); err != nil {
		t.Fatalf("RegisterIncomingRequest = %v", err)
	}

	waitNext, err = r.RegisterResponse(2, 1)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	assertErrorIs(t, err, ErrNotResponse)

	req, _ := r.Find(2)
	if req.ActualAnswers != 1 {
		t.Fatal("RegisterResponse on incoming mutated ActualAnswers")
	}

	id, err := r.RegisterOutcomingRequest(1)
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest = %v", err)
	}

	waitNext, err = r.RegisterResponse(id, 2)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	got := asError[ErrorUnexpectedAnswers](t, err)
	if got.Expected() != 1 || got.Got() != 2 {
		t.Fatalf("error = %+v, want expected 1 got 2", got)
	}

	req, _ = r.Find(id)
	if req.ActualAnswers != 0 {
		t.Fatalf("failed RegisterResponse mutated ActualAnswers to %d", req.ActualAnswers)
	}
}

func TestRegisterResponseZeroGotOnPartial(t *testing.T) {
	r := readyRequestsV1(t)
	id, err := r.RegisterOutcomingRequest(2)
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest = %v", err)
	}

	waitNext, err := r.RegisterResponse(id, 0)
	if err != nil || !waitNext {
		t.Fatalf("RegisterResponse(0) = (%v, %v), want wait", waitNext, err)
	}

	req, _ := r.Find(id)
	if req.ActualAnswers != 0 {
		t.Fatalf("ActualAnswers = %d, want 0", req.ActualAnswers)
	}
}

func TestRegisterResponseAfterTerminate(t *testing.T) {
	r := readyRequestsV1(t)
	id, err := r.RegisterOutcomingRequest(1)
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest = %v", err)
	}
	r.Terminate(id)

	waitNext, err := r.RegisterResponse(id, 1)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	assertErrorIs(t, err, ErrRequestNotFound)
}

func TestFindReturnsCopy(t *testing.T) {
	r := readyRequestsV1(t)
	id, err := r.RegisterOutcomingRequest(2)
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest = %v", err)
	}

	got, ok := r.Find(id)
	if !ok {
		t.Fatal("Find(id) = false")
	}
	got.ActualAnswers = 99
	got.IsOutcoming = false

	stored, ok := r.Find(id)
	if !ok || stored.ActualAnswers != 0 || !stored.IsOutcoming {
		t.Fatalf("table mutated through Find copy: %+v", stored)
	}
}

func TestTerminate(t *testing.T) {
	r := readyRequestsV1(t)

	r.Terminate(1)

	id, err := r.RegisterOutcomingRequest(1)
	if err != nil || id != 1 {
		t.Fatalf("RegisterOutcomingRequest = (%v, %v)", id, err)
	}
	if err := r.RegisterIncomingRequest(9); err != nil {
		t.Fatalf("RegisterIncomingRequest = %v", err)
	}

	r.Terminate(id)
	r.Terminate(id)
	if _, ok := r.Find(id); ok {
		t.Fatal("Find(id) after Terminate = true")
	}
	if _, ok := r.Find(9); !ok {
		t.Fatal("Terminate removed an unrelated id")
	}

	id, err = r.RegisterOutcomingRequest(1)
	if err != nil || id != 2 {
		t.Fatalf("re-register after Terminate = (%v, %v), want 2", id, err)
	}
}

func TestTerminateAll(t *testing.T) {
	r := readyRequestsV1(t)

	r.TerminateAll()

	for _, id := range []fields.RequestID{1, 2, 3} {
		if err := r.RegisterIncomingRequest(id); err != nil {
			t.Fatalf("RegisterIncomingRequest(%d) = %v", id, err)
		}
	}

	r.TerminateAll()

	for _, id := range []fields.RequestID{1, 2, 3} {
		if _, ok := r.Find(id); ok {
			t.Fatalf("Find(%d) after TerminateAll = true", id)
		}
	}
	if !r.IsGranted() || !r.IsActive() {
		t.Fatalf("TerminateAll cleared flags: granted=%v active=%v", r.IsGranted(), r.IsActive())
	}

	if err := r.RegisterIncomingRequest(1); err != nil {
		t.Fatalf("RegisterIncomingRequest after TerminateAll = %v", err)
	}
}

func TestConcurrentOutcomingRegister(t *testing.T) {
	const n = 64

	r := readyRequestsV1(t)
	ids := make([]fields.RequestID, n)
	var wg sync.WaitGroup

	for i := range n {
		wg.Go(func() {
			id, err := r.RegisterOutcomingRequest(1)
			if err != nil || id == 0 {
				t.Errorf("RegisterOutcomingRequest = (%v, %v)", id, err)
				return
			}
			ids[i] = id
		})
	}
	wg.Wait()

	seen := make(map[fields.RequestID]struct{}, n)
	for _, id := range ids {
		if id == 0 {
			t.Fatal("missing id")
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = struct{}{}

		req, ok := r.Find(id)
		if !ok || !req.IsOutcoming {
			t.Fatalf("Find(%d) = (%+v, %v)", id, req, ok)
		}
	}
	if len(seen) != n {
		t.Fatalf("unique ids = %d, want %d", len(seen), n)
	}
}

func TestConcurrentIncomingDuplicate(t *testing.T) {
	const n = 32

	r := readyRequestsV1(t)
	var wg sync.WaitGroup
	var success, conflict atomic.Int32

	for range n {
		wg.Go(func() {
			err := r.RegisterIncomingRequest(1)
			if err == nil {
				success.Add(1)
				return
			}
			if _, ok := errors.AsType[ErrorRequestIDUsed](err); !ok {
				t.Errorf("RegisterIncomingRequest = %v, want ErrorRequestIDUsed", err)
				return
			}
			conflict.Add(1)
		})
	}
	wg.Wait()

	if success.Load() != 1 {
		t.Errorf("success = %d, want 1", success.Load())
	}
	if conflict.Load() != n-1 {
		t.Errorf("conflict = %d, want %d", conflict.Load(), n-1)
	}
}

func TestConcurrentRegisterResponse(t *testing.T) {
	const n = 32

	r := readyRequestsV1(t)
	id, err := r.RegisterOutcomingRequest(uint32(n))
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest = %v", err)
	}

	var wg sync.WaitGroup
	var okCount atomic.Int32

	for range n {
		wg.Go(func() {
			_, err := r.RegisterResponse(id, 1)
			if err != nil {
				t.Errorf("RegisterResponse = %v", err)
				return
			}
			okCount.Add(1)
		})
	}
	wg.Wait()

	if okCount.Load() != n {
		t.Fatalf("ok = %d, want %d", okCount.Load(), n)
	}

	req, ok := r.Find(id)
	if !ok || req.ActualAnswers != uint32(n) {
		t.Fatalf("request = %+v ok=%v, want ActualAnswers %d", req, ok, n)
	}
}

func TestConcurrentGrantActivateFind(t *testing.T) {
	r := readyRequestsV1(t)
	id, err := r.RegisterOutcomingRequest(1)
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest = %v", err)
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		for range 100 {
			r.Grant(true)
			_ = r.IsGranted()
		}
	})
	wg.Go(func() {
		for range 100 {
			r.Activate(true)
			_ = r.IsActive()
		}
	})
	wg.Go(func() {
		for range 100 {
			_, _ = r.Find(id)
			_ = r.IsGranted()
			_ = r.IsActive()
		}
	})

	wg.Wait()

	if _, ok := r.Find(id); !ok {
		t.Fatal("request lost during concurrent flag updates")
	}
}

func TestNilRequestsV1Panics(t *testing.T) {
	var r *RequestsV1

	tests := []struct {
		name string
		call func()
	}{
		{"grant", func() { r.Grant(true) }},
		{"activate", func() { r.Activate(true) }},
		{"register outcoming", func() { _, _ = r.RegisterOutcomingRequest(1) }},
		{"register incoming", func() { _ = r.RegisterIncomingRequest(1) }},
		{"register response", func() { _, _ = r.RegisterResponse(1, 1) }},
		{"terminate", func() { r.Terminate(1) }},
		{"terminate all", func() { r.TerminateAll() }},
		{"find", func() { _, _ = r.Find(1) }},
		{"is granted", func() { _ = r.IsGranted() }},
		{"is active", func() { _ = r.IsActive() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			tt.call()
		})
	}
}

func TestNilRegisterIncomingProhibitedNoPanic(t *testing.T) {
	var r *RequestsV1

	err := r.RegisterIncomingRequest(PROHIBITED_REQUEST_ID)
	assertErrorIs(t, err, ErrProhibitedRequestID)
}

func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("error = %v, want %v", err, target)
	}
}

func asError[T error](t *testing.T, err error) T {
	t.Helper()
	got, ok := errors.AsType[T](err)
	if !ok {
		t.Fatalf("error = %v, want %T", err, *new(T))
	}
	return got
}
