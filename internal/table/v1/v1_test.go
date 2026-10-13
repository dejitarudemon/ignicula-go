package table_v1

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func newCaller() ResponseV1Chan {
	return make(ResponseV1Chan, 8)
}

func readyRequestsV1(t *testing.T) *RequestsV1 {
	t.Helper()
	r := NewRequestsV1()
	r.Grant(true)
	r.Activate(true)
	return r
}

func TestNewRequestV1ClampsExpected(t *testing.T) {
	caller := newCaller()
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
			got := NewRequestV1(tt.expected, fields.Read, caller)
			if got.Command != fields.Read {
				t.Fatalf("Command = %v, want Read", got.Command)
			}
			if got.ToCaller == nil {
				t.Fatal("ToCaller = nil")
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
	caller := newCaller()
	got := NewRequestV1WithActual(0, fields.Ping, caller, 9)
	if got.Command != fields.Ping {
		t.Fatalf("Command = %v, want Ping", got.Command)
	}
	if got.ExpectedAnswers != 1 {
		t.Fatalf("ExpectedAnswers = %d, want 1", got.ExpectedAnswers)
	}
	if got.ActualAnswers != 9 {
		t.Fatalf("ActualAnswers = %d, want 9", got.ActualAnswers)
	}
}

func TestRequestV1Add(t *testing.T) {
	r := NewRequestV1(3, fields.Read, newCaller())

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
	r := NewRequestV1(2, fields.Read, newCaller())
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
	r := NewRequestV1WithActual(1, fields.Read, newCaller(), 2)

	full, err := r.add(0)
	if full {
		t.Fatal("full = true, want false")
	}
	assertErrorIs(t, err, ErrInvalidActualAnswersState)
}

func TestRequestV1AddOnAlreadyFull(t *testing.T) {
	r := NewRequestV1WithActual(2, fields.Read, newCaller(), 2)

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
	id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil || id != 1 {
		t.Fatalf("RegisterRequest = (%v, %v)", id, err)
	}
	r.Grant(false)
	r.Activate(false)
	r.Grant(true)
	r.Activate(true)
	if _, ok := r.Find(id); !ok {
		t.Fatal("Grant/Activate cleared the request map")
	}
}

func TestRegisterRequestNilChan(t *testing.T) {
	r := NewRequestsV1()
	id, err := r.RegisterRequest(fields.Read, nil, 1)
	if id != 0 {
		t.Fatalf("id = %v, want 0", id)
	}
	assertErrorIs(t, err, ErrNilResponseV1Chan)

	r = readyRequestsV1(t)
	id, err = r.RegisterRequest(fields.Read, nil, 1)
	if id != 0 {
		t.Fatalf("id = %v, want 0", id)
	}
	assertErrorIs(t, err, ErrNilResponseV1Chan)
}

func TestRegisterRequiresGrantedAndActive(t *testing.T) {
	r := NewRequestsV1()
	caller := newCaller()

	id, err := r.RegisterRequest(fields.Read, caller, 1)
	if id != 0 {
		t.Fatalf("id = %v, want 0", id)
	}
	assertErrorIs(t, err, ErrVersionNeitherGrantedNorActivated)

	waitNext, err := r.RegisterResponse(1, 1)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	assertErrorIs(t, err, ErrVersionNeitherGrantedNorActivated)

	r.Grant(true)
	_, err = r.RegisterRequest(fields.Read, caller, 1)
	assertErrorIs(t, err, ErrVersionNeitherGrantedNorActivated)

	r.Grant(false)
	r.Activate(true)
	_, err = r.RegisterRequest(fields.Read, caller, 1)
	assertErrorIs(t, err, ErrVersionNeitherGrantedNorActivated)

	r.Grant(true)
	id, err = r.RegisterRequest(fields.Read, caller, 1)
	if err != nil || id != 1 {
		t.Fatalf("with both flags = (%v, %v)", id, err)
	}
}

func TestRegisterRequestAllocatesNonZero(t *testing.T) {
	r := readyRequestsV1(t)
	caller := newCaller()

	id, err := r.RegisterRequest(fields.Read, caller, 0)
	if err != nil || id != 1 {
		t.Fatalf("RegisterRequest(0) = (%v, %v), want (1, nil)", id, err)
	}

	req, ok := r.Find(id)
	if !ok {
		t.Fatal("Find(id) = false")
	}
	if req.Command != fields.Read || req.ExpectedAnswers != 1 || req.ActualAnswers != 0 {
		t.Fatalf("request = %+v, want Read expecting 1", req)
	}
	if req.ToCaller == nil {
		t.Fatal("ToCaller = nil")
	}
	if _, ok := r.Find(PROHIBITED_REQUEST_ID); ok {
		t.Fatal("prohibited id 0 was allocated")
	}

	id, err = r.RegisterRequest(fields.Batch, caller, 5)
	if err != nil || id != 2 {
		t.Fatalf("RegisterRequest(5) = (%v, %v), want (2, nil)", id, err)
	}

	req, ok = r.Find(id)
	if !ok {
		t.Fatal("Find(2) = false")
	}
	if req.Command != fields.Batch || req.ExpectedAnswers != 5 {
		t.Fatalf("request = %+v, want Batch expecting 5", req)
	}
}

func TestRegisterRequestSkipsOccupied(t *testing.T) {
	r := readyRequestsV1(t)
	caller := newCaller()

	r.requests[1] = NewRequestV1(1, fields.Ping, caller)
	r.requests[3] = NewRequestV1(1, fields.Ping, caller)

	id, err := r.RegisterRequest(fields.Read, caller, 1)
	if err != nil || id != 2 {
		t.Fatalf("RegisterRequest = (%v, %v), want (2, nil)", id, err)
	}

	id, err = r.RegisterRequest(fields.Read, caller, 1)
	if err != nil || id != 4 {
		t.Fatalf("RegisterRequest again = (%v, %v), want (4, nil)", id, err)
	}
}

func TestRegisterRequestWraps(t *testing.T) {
	r := readyRequestsV1(t)
	r.cursor = math.MaxUint32

	id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil || id != 1 {
		t.Fatalf("RegisterRequest at max = (%v, %v), want (1, nil)", id, err)
	}
	if _, ok := r.Find(PROHIBITED_REQUEST_ID); ok {
		t.Fatal("wrap must skip prohibited id 0")
	}
}

func TestRegisterRequestReusesAfterTerminate(t *testing.T) {
	r := readyRequestsV1(t)

	id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil || id != 1 {
		t.Fatalf("first register = (%v, %v)", id, err)
	}
	r.Terminate(id)
	r.cursor = math.MaxUint32

	id, err = r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil || id != 1 {
		t.Fatalf("register at max = (%v, %v), want reused 1", id, err)
	}
}

func TestIncreaseSkipsProhibitedAndBusy(t *testing.T) {
	r := NewRequestsV1()
	r.cursor = math.MaxUint32
	r.requests[1] = NewRequestV1(1, fields.Read, newCaller())

	if err := r.increase(); err != nil {
		t.Fatalf("increase = %v", err)
	}
	if r.cursor != 2 {
		t.Fatalf("cursor = %d, want 2 after skipping 0 and busy 1", r.cursor)
	}
}

func TestRegisterResponse(t *testing.T) {
	r := readyRequestsV1(t)

	id, err := r.RegisterRequest(fields.Read, newCaller(), 3)
	if err != nil || id != 1 {
		t.Fatalf("RegisterRequest = (%v, %v)", id, err)
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

	id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil {
		t.Fatalf("RegisterRequest = %v", err)
	}

	waitNext, err = r.RegisterResponse(id, 2)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	got := asError[ErrorUnexpectedAnswers](t, err)
	if got.Expected() != 1 || got.Got() != 2 {
		t.Fatalf("error = %+v, want expected 1 got 2", got)
	}

	req, _ := r.Find(id)
	if req.ActualAnswers != 0 {
		t.Fatalf("failed RegisterResponse mutated ActualAnswers to %d", req.ActualAnswers)
	}
}

func TestRegisterResponseZeroGotOnPartial(t *testing.T) {
	r := readyRequestsV1(t)
	id, err := r.RegisterRequest(fields.Read, newCaller(), 2)
	if err != nil {
		t.Fatalf("RegisterRequest = %v", err)
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
	id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil {
		t.Fatalf("RegisterRequest = %v", err)
	}
	r.Terminate(id)

	waitNext, err := r.RegisterResponse(id, 1)
	if waitNext {
		t.Fatal("waitNext = true, want false")
	}
	assertErrorIs(t, err, ErrRequestNotFound)
}

func TestFindReturnsShallowCopy(t *testing.T) {
	r := readyRequestsV1(t)
	caller := newCaller()
	id, err := r.RegisterRequest(fields.Write, caller, 2)
	if err != nil {
		t.Fatalf("RegisterRequest = %v", err)
	}

	got, ok := r.Find(id)
	if !ok {
		t.Fatal("Find(id) = false")
	}
	got.ActualAnswers = 99
	got.Command = fields.Delete

	stored, ok := r.Find(id)
	if !ok || stored.ActualAnswers != 0 || stored.Command != fields.Write {
		t.Fatalf("table mutated through Find copy: %+v", stored)
	}
	if stored.ToCaller == nil || stored.ToCaller != caller {
		t.Fatal("ToCaller not shared with stored row")
	}
}

func TestTerminate(t *testing.T) {
	r := readyRequestsV1(t)

	r.Terminate(1)

	id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil || id != 1 {
		t.Fatalf("RegisterRequest = (%v, %v)", id, err)
	}
	other, err := r.RegisterRequest(fields.Ping, newCaller(), 1)
	if err != nil || other != 2 {
		t.Fatalf("second RegisterRequest = (%v, %v)", other, err)
	}

	r.Terminate(id)
	r.Terminate(id)
	if _, ok := r.Find(id); ok {
		t.Fatal("Find(id) after Terminate = true")
	}
	if _, ok := r.Find(other); !ok {
		t.Fatal("Terminate removed an unrelated id")
	}

	id, err = r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil || id != 3 {
		t.Fatalf("re-register after Terminate = (%v, %v), want 3", id, err)
	}
}

func TestTerminateAll(t *testing.T) {
	r := readyRequestsV1(t)

	r.TerminateAll()

	for range 3 {
		if _, err := r.RegisterRequest(fields.Read, newCaller(), 1); err != nil {
			t.Fatalf("RegisterRequest = %v", err)
		}
	}

	r.TerminateAll()

	for id := fields.RequestID(1); id <= 3; id++ {
		if _, ok := r.Find(id); ok {
			t.Fatalf("Find(%d) after TerminateAll = true", id)
		}
	}
	if !r.IsGranted() || !r.IsActive() {
		t.Fatalf("TerminateAll cleared flags: granted=%v active=%v", r.IsGranted(), r.IsActive())
	}

	if _, err := r.RegisterRequest(fields.Read, newCaller(), 1); err != nil {
		t.Fatalf("RegisterRequest after TerminateAll = %v", err)
	}
}

func TestConcurrentRegisterRequest(t *testing.T) {
	const n = 64

	r := readyRequestsV1(t)
	ids := make([]fields.RequestID, n)
	var wg sync.WaitGroup

	for i := range n {
		wg.Go(func() {
			id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
			if err != nil || id == 0 {
				t.Errorf("RegisterRequest = (%v, %v)", id, err)
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
		if !ok || req.Command != fields.Read {
			t.Fatalf("Find(%d) = (%+v, %v)", id, req, ok)
		}
	}
	if len(seen) != n {
		t.Fatalf("unique ids = %d, want %d", len(seen), n)
	}
}

func TestConcurrentRegisterResponse(t *testing.T) {
	const n = 32

	r := readyRequestsV1(t)
	id, err := r.RegisterRequest(fields.Batch, newCaller(), uint32(n))
	if err != nil {
		t.Fatalf("RegisterRequest = %v", err)
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
	id, err := r.RegisterRequest(fields.Read, newCaller(), 1)
	if err != nil {
		t.Fatalf("RegisterRequest = %v", err)
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
		{"register", func() { _, _ = r.RegisterRequest(fields.Read, newCaller(), 1) }},
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
