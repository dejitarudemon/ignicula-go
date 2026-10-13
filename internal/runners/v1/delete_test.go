package runner_v1

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func TestDeleteSuccess(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Delete = time.Second

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Delete(ctx, []byte("key"))
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode delete = %v", err)
	}
	if frm.Body.Command() != fields.Delete {
		t.Fatalf("command = %v, want Delete", frm.Body.Command())
	}
	db, ok := frm.Body.(bodies.Delete)
	if !ok || string(db) != "key" {
		t.Fatalf("body = %#v", frm.Body)
	}

	answer, err := mustBuilder(t).NewDeleteAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewDeleteAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, answer)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Delete = %v", err)
	}
}

func TestDeleteServerError(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Delete = time.Second
	runner.timeouts.MaxAttempts = 1

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Delete(ctx, []byte("key"))
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode delete = %v", err)
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
		t.Fatalf("Delete = %v (%T), want ServerError", err, err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "Unauthorized") {
		t.Fatalf("Error() = %q, want readable Unauthorized", msg)
	}
}

func TestDeleteRetriesRequestsConflict(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Delete = time.Second
	runner.timeouts.MaxAttempts = 2
	runner.timeouts.Backoff = 0

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Delete(ctx, []byte("key"))
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode = %v", err)
	}
	fail, err := mustBuilder(t).NewErrAnswer(frm.RequestID, protocolerrs.NewErrorRequestsConflict(frm.RequestID))
	if err != nil {
		t.Fatalf("NewErrAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, fail)); err != nil {
		t.Fatalf("Consume fail = %v", err)
	}

	raw = <-requests
	frm, err = runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode retry = %v", err)
	}
	okAnswer, err := mustBuilder(t).NewDeleteAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewDeleteAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, okAnswer)); err != nil {
		t.Fatalf("Consume ok = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Delete after retry = %v", err)
	}
}

func TestDeleteActivating(t *testing.T) {
	runner, _, _ := newActivatingRunner(t)
	err := runner.Delete(context.Background(), []byte("key"))
	var internal runners.InternalError
	if !errors.As(err, &internal) || !errors.Is(err, runners.ErrActivating) {
		t.Fatalf("Delete = %v, want ErrActivating", err)
	}
}
