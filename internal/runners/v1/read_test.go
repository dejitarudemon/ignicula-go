package runner_v1

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	"github.com/dejitarudemon/ignicula-go/values"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	protovalues "github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func TestReadSuccess(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Read = time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		val values.Value
		err error
	}
	done := make(chan result, 1)
	go func() {
		val, err := runner.Read(ctx, []byte("key"))
		done <- result{val: val, err: err}
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode read = %v", err)
	}
	if frm.Body.Command() != fields.Read {
		t.Fatalf("command = %v, want Read", frm.Body.Command())
	}

	answer, err := mustBuilder(t).NewReadAnswer(frm.RequestID, protovalues.String("hello"))
	if err != nil {
		t.Fatalf("NewReadAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, answer)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Read = %v", got.err)
		}
		sv, ok := got.val.(values.StringValue)
		if !ok || string(sv) != "hello" {
			t.Fatalf("Read value = %#v, want StringValue(hello)", got.val)
		}
	case <-time.After(time.Second):
		t.Fatal("Read did not return")
	}
}

func TestReadServerError(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Read = time.Second
	runner.timeouts.MaxAttempts = 1

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		_, err := runner.Read(ctx, []byte("key"))
		done <- err
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode read = %v", err)
	}
	answer, err := mustBuilder(t).NewErrAnswer(frm.RequestID, protocolerrs.NewErrorUnauthorized([]byte("x")))
	if err != nil {
		t.Fatalf("NewErrAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, answer)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	err = <-done
	var serverErr runners.ServerError
	if !errors.As(err, &serverErr) {
		t.Fatalf("Read = %v (%T), want ServerError", err, err)
	}
}

func TestReadRetriesRequestsConflict(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Read = time.Second
	runner.timeouts.MaxAttempts = 2
	runner.timeouts.Backoff = 0

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		_, err := runner.Read(ctx, []byte("key"))
		done <- err
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode first read = %v", err)
	}
	fail, err := mustBuilder(t).NewErrAnswer(frm.RequestID, protocolerrs.NewErrorRequestsConflict(frm.RequestID))
	if err != nil {
		t.Fatalf("NewErrAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, fail)); err != nil {
		t.Fatalf("Consume conflict = %v", err)
	}

	raw = <-requests
	frm, err = runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode retry read = %v", err)
	}
	okAnswer, err := mustBuilder(t).NewReadAnswer(frm.RequestID, protovalues.Bytes([]byte{1}))
	if err != nil {
		t.Fatalf("NewReadAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, okAnswer)); err != nil {
		t.Fatalf("Consume ok = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Read after retry = %v", err)
	}
}

func TestReadTimeout(t *testing.T) {
	runner, table, requests := newTestRunner(t)
	runner.timeouts.Read = 20 * time.Millisecond
	runner.timeouts.MaxAttempts = 1

	go func() {
		<-requests
	}()

	_, err := runner.Read(context.Background(), []byte("key"))
	var internal runners.InternalError
	if !errors.As(err, &internal) {
		t.Fatalf("Read = %v (%T), want InternalError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Read = %v, want deadline exceeded", err)
	}
	if _, ok := table.Find(1); ok {
		t.Fatal("timed-out read request still registered")
	}
}

func TestReadActivating(t *testing.T) {
	runner, _, _ := newActivatingRunner(t)
	_, err := runner.Read(context.Background(), []byte("key"))
	var internal runners.InternalError
	if !errors.As(err, &internal) || !errors.Is(err, runners.ErrActivating) {
		t.Fatalf("Read = %v, want ErrActivating", err)
	}
}
