package runner_v1

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func TestPingSuccess(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Ping = time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runner.Ping(ctx)
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode ping = %v", err)
	}
	if frm.Body.Command() != fields.Ping {
		t.Fatalf("command = %v, want Ping", frm.Body.Command())
	}

	answer, err := mustBuilder(t).NewPingAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewPingAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, answer)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Ping = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Ping did not return")
	}
}

func TestPingServerError(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Ping = time.Second
	runner.timeouts.MaxAttempts = 1

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Ping(ctx)
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode ping = %v", err)
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
		t.Fatalf("Ping = %v (%T), want ServerError", err, err)
	}
	var unauthorized protocolerrs.ErrorUnauthorized
	if !errors.As(err, &unauthorized) {
		t.Fatalf("Unwrap = %v, want ErrorUnauthorized", err)
	}
}

func TestPingRetriesRequestsConflict(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Ping = time.Second
	runner.timeouts.MaxAttempts = 2
	runner.timeouts.Backoff = 0

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Ping(ctx)
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode first ping = %v", err)
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
		t.Fatalf("decode retry ping = %v", err)
	}
	okAnswer, err := mustBuilder(t).NewPingAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewPingAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, okAnswer)); err != nil {
		t.Fatalf("Consume ok = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Ping after retry = %v", err)
	}
}

func TestPingProtocolViolation(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Ping = time.Second

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Ping(ctx)
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode ping = %v", err)
	}

	wrong, err := mustBuilder(t).NewWriteAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewWriteAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, wrong)); err != nil {
		assertLogAndIgnore(t, err)
	}

	err = <-done
	var violation runners.ProtocolViolationByServer
	if !errors.As(err, &violation) {
		t.Fatalf("Ping = %v (%T), want ProtocolViolationByServer", err, err)
	}
}

func TestPingTimeout(t *testing.T) {
	runner, table, requests := newTestRunner(t)
	runner.timeouts.Ping = 20 * time.Millisecond
	runner.timeouts.MaxAttempts = 1

	go func() {
		<-requests
	}()

	err := runner.Ping(context.Background())
	var internal runners.InternalError
	if !errors.As(err, &internal) {
		t.Fatalf("Ping = %v (%T), want InternalError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Ping = %v, want deadline exceeded", err)
	}

	if _, ok := table.Find(1); ok {
		t.Fatal("timed-out ping request still registered")
	}
}

func TestConsumeErrorAnswerAccepted(t *testing.T) {
	runner, table, _ := newTestRunner(t)
	fb := mustBuilder(t)

	id, caller := registerRequest(t, table, fields.Ping, 1)

	frm, err := fb.NewErrAnswer(id, protocolerrs.NewErrorUnauthorized([]byte("x")))
	if err != nil {
		t.Fatalf("NewErrAnswer = %v", err)
	}

	if err := runner.Consume(context.Background(), frameReader(t, frm)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	resp, ok := <-caller
	if !ok {
		t.Fatal("caller closed before response")
	}
	if resp.Err != nil {
		t.Fatalf("response err = %v, want nil (ErrorAnswer is a valid body)", resp.Err)
	}
	if _, ok := <-caller; ok {
		t.Fatal("caller channel should be closed after ErrorAnswer")
	}
	if _, ok := table.Find(id); ok {
		t.Fatal("request still registered after ErrorAnswer")
	}
}
