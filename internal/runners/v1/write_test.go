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
	"github.com/dejitarudemon/ignicula-go/values"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func TestWriteSuccess(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Write = time.Second

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Write(ctx, []byte("key"), values.StringValue("hello"))
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode write = %v", err)
	}
	if frm.Body.Command() != fields.Write {
		t.Fatalf("command = %v, want Write", frm.Body.Command())
	}
	wb, ok := frm.Body.(bodies.Write)
	if !ok || string(wb.Key) != "key" {
		t.Fatalf("body = %#v", frm.Body)
	}

	answer, err := mustBuilder(t).NewWriteAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewWriteAnswer = %v", err)
	}
	if err := runner.Consume(ctx, frameReader(t, answer)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Write = %v", err)
	}
}

func TestWriteServerError(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	runner.timeouts.Write = time.Second
	runner.timeouts.MaxAttempts = 1

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- runner.Write(ctx, []byte("key"), values.BytesValue{1, 2})
	}()

	raw := <-requests
	frm, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("decode write = %v", err)
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
		t.Fatalf("Write = %v (%T), want ServerError", err, err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "Unauthorized") {
		t.Fatalf("Error() = %q, want readable Unauthorized", msg)
	}
}

func TestWriteUnsupportedValue(t *testing.T) {
	runner, _, _ := newTestRunner(t)
	err := runner.Write(context.Background(), []byte("k"), unsupportedValue{})
	var internal runners.InternalError
	if !errors.As(err, &internal) {
		t.Fatalf("Write = %v, want InternalError", err)
	}
}

type unsupportedValue struct{}

func (unsupportedValue) Type() values.Type { return 255 }
