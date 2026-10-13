package runner_v1

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/internal/runners"
	"github.com/dejitarudemon/ignicula-go/internal/table/v1"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/builder"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/frame"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func testRuntimeConfig() configs.RunnerV1Config {
	return configs.RunnerV1Config{
		MaxResponseSize: 1 << 20,
		MaxRequestSize:  1 << 20,
		MaxBatchSize:    64,
	}
}

func newTestRunner(t *testing.T) (*Runner, *table_v1.RequestsV1, chan []byte) {
	t.Helper()

	table := table_v1.NewRequestsV1()
	table.Grant(true)
	table.Activate(true)

	requests := make(chan []byte, 8)
	runner, err := NewRunner(table, requests, testRuntimeConfig())
	if err != nil {
		t.Fatalf("NewRunner = %v", err)
	}

	return runner, table, requests
}

func encodeFrame(t *testing.T, frm frame.Frame) []byte {
	t.Helper()

	buf := buffer.Slice{}
	buf.Preallocate(frm.Size())
	if err := frm.Encode(&buf, nil); err != nil {
		t.Fatalf("Encode = %v", err)
	}
	return buf.Bytes()
}

func frameReader(t *testing.T, frm frame.Frame) *bufio.Reader {
	t.Helper()
	return bufio.NewReader(bytes.NewReader(encodeFrame(t, frm)))
}

func registerRequest(t *testing.T, table *table_v1.RequestsV1, command fields.Command, expected uint32) (fields.RequestID, table_v1.ResponseV1Chan) {
	t.Helper()

	ch := make(table_v1.ResponseV1Chan, 8)
	id, err := table.RegisterRequest(command, ch, expected)
	if err != nil {
		t.Fatalf("RegisterRequest = %v", err)
	}
	return id, ch
}

func mustBuilder(t *testing.T) builder.FrameBuilder {
	t.Helper()
	return builder.NewFrameBuilder(1 << 20)
}

func assertCloseConnection(t *testing.T, err error) runners.CloseConnection {
	t.Helper()
	got, ok := errors.AsType[runners.CloseConnection](err)
	if !ok {
		t.Fatalf("error = %v (%T), want CloseConnection", err, err)
	}
	return got
}

func assertLogAndIgnore(t *testing.T, err error) runners.LogAndIgnore {
	t.Helper()
	got, ok := errors.AsType[runners.LogAndIgnore](err)
	if !ok {
		t.Fatalf("error = %v (%T), want LogAndIgnore", err, err)
	}
	return got
}

func TestConsumeDecodeError(t *testing.T) {
	runner, _, _ := newTestRunner(t)

	err := runner.Consume(context.Background(), bufio.NewReader(bytes.NewReader([]byte{0x01})))
	closeErr := assertCloseConnection(t, err)
	if closeErr.Unwrap() == nil {
		t.Fatalf("CloseConnection cause = nil, error = %v", err)
	}
}

func TestConsumeUnsolicitedNonPingIgnored(t *testing.T) {
	runner, _, _ := newTestRunner(t)
	fb := mustBuilder(t)

	frm, err := fb.NewRead(1, fields.Key("key"))
	if err != nil {
		t.Fatalf("NewRead = %v", err)
	}

	err = runner.Consume(context.Background(), frameReader(t, frm))
	ignoreErr := assertLogAndIgnore(t, err)

	var violation runners.ProtocolViolationByServer
	if !errors.As(ignoreErr, &violation) {
		t.Fatalf("cause = %v, want ProtocolViolationByServer", ignoreErr.Unwrap())
	}
}

func TestConsumeUnsolicitedPingReplies(t *testing.T) {
	runner, table, requests := newTestRunner(t)
	fb := mustBuilder(t)

	frm, err := fb.NewPing(7)
	if err != nil {
		t.Fatalf("NewPing = %v", err)
	}

	if err := runner.Consume(context.Background(), frameReader(t, frm)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	select {
	case raw := <-requests:
		got, err := runner.decoder.DecodeFrame(bufio.NewReader(bytes.NewReader(raw)))
		if err != nil {
			t.Fatalf("decode reply = %v", err)
		}
		if got.RequestID != 7 {
			t.Fatalf("reply id = %d, want 7", got.RequestID)
		}
		if _, ok := got.Body.(bodies.PingAnswer); !ok {
			t.Fatalf("body = %T, want PingAnswer", got.Body)
		}
	default:
		t.Fatal("expected ping answer on requests chan")
	}

	if _, ok := table.Find(7); ok {
		t.Fatal("ping id still registered after reply")
	}
}

func TestConsumeUnsolicitedPingCancelled(t *testing.T) {
	runner, _, requests := newTestRunner(t)
	fb := mustBuilder(t)

	runner.requests = make(chan []byte)

	frm, err := fb.NewPing(3)
	if err != nil {
		t.Fatalf("NewPing = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runner.Consume(ctx, frameReader(t, frm))
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assertCloseConnection(t, err)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Consume did not return after cancel")
	}

	select {
	case <-requests:
		t.Fatal("unexpected write on original requests chan")
	default:
	}
}

func TestConsumeAnswerCompletes(t *testing.T) {
	runner, table, _ := newTestRunner(t)
	fb := mustBuilder(t)

	id, caller := registerRequest(t, table, fields.Read, 1)

	frm, err := fb.NewReadAnswer(id, values.String("ok"))
	if err != nil {
		t.Fatalf("NewReadAnswer = %v", err)
	}

	if err := runner.Consume(context.Background(), frameReader(t, frm)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	resp, ok := <-caller
	if !ok {
		t.Fatal("caller closed before response")
	}
	if resp.Err != nil {
		t.Fatalf("response err = %v", resp.Err)
	}
	if resp.Response.RequestID != id {
		t.Fatalf("response id = %d, want %d", resp.Response.RequestID, id)
	}

	if _, ok := <-caller; ok {
		t.Fatal("caller channel should be closed after completion")
	}
	if _, ok := table.Find(id); ok {
		t.Fatal("request still registered after completion")
	}
}

func TestConsumeAnswerPartialKeepsOpen(t *testing.T) {
	runner, table, _ := newTestRunner(t)
	fb := mustBuilder(t)

	id, caller := registerRequest(t, table, fields.Write, 2)

	frm, err := fb.NewWriteAnswer(id)
	if err != nil {
		t.Fatalf("NewWriteAnswer = %v", err)
	}

	if err := runner.Consume(context.Background(), frameReader(t, frm)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	resp, ok := <-caller
	if !ok {
		t.Fatal("caller closed after partial answer")
	}
	if resp.Err != nil {
		t.Fatalf("response err = %v", resp.Err)
	}

	select {
	case <-caller:
		t.Fatal("caller received extra value or closed too early")
	default:
	}

	req, ok := table.Find(id)
	if !ok {
		t.Fatal("request terminated after partial answer")
	}
	if req.ActualAnswers != 1 {
		t.Fatalf("ActualAnswers = %d, want 1", req.ActualAnswers)
	}
}

func TestConsumeAnswerBatchCountsNested(t *testing.T) {
	runner, table, _ := newTestRunner(t)
	fb := mustBuilder(t)

	id, caller := registerRequest(t, table, fields.Batch, 2)

	frm, err := fb.NewBatchAnswer(id, *builder.NewBatchResultsBuilder().AddWrite(1).AddDelete(2))
	if err != nil {
		t.Fatalf("NewBatchAnswer = %v", err)
	}

	if err := runner.Consume(context.Background(), frameReader(t, frm)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	resp, ok := <-caller
	if !ok {
		t.Fatal("caller closed before response")
	}
	if resp.Err != nil {
		t.Fatalf("response err = %v", resp.Err)
	}
	if _, ok := <-caller; ok {
		t.Fatal("caller channel should be closed after batch completion")
	}
	if _, ok := table.Find(id); ok {
		t.Fatal("batch request still registered")
	}
}

func TestConsumeNotAnswerFailsRequest(t *testing.T) {
	runner, table, _ := newTestRunner(t)
	fb := mustBuilder(t)

	id, caller := registerRequest(t, table, fields.Read, 1)

	frm, err := fb.NewPing(id)
	if err != nil {
		t.Fatalf("NewPing = %v", err)
	}

	err = runner.Consume(context.Background(), frameReader(t, frm))
	assertLogAndIgnore(t, err)

	resp, ok := <-caller
	if !ok {
		t.Fatal("caller closed before failure response")
	}
	if resp.Err == nil {
		t.Fatal("response err = nil, want protocol violation")
	}
	var violation runners.ProtocolViolationByServer
	if !errors.As(resp.Err, &violation) {
		t.Fatalf("response err = %v, want ProtocolViolationByServer", resp.Err)
	}
	if _, ok := <-caller; ok {
		t.Fatal("caller channel should be closed after failure")
	}
	if _, ok := table.Find(id); ok {
		t.Fatal("request still registered after failure")
	}
}

func TestConsumeWrongAnswerCommandFailsRequest(t *testing.T) {
	runner, table, _ := newTestRunner(t)
	fb := mustBuilder(t)

	id, caller := registerRequest(t, table, fields.Read, 1)

	frm, err := fb.NewWriteAnswer(id)
	if err != nil {
		t.Fatalf("NewWriteAnswer = %v", err)
	}

	err = runner.Consume(context.Background(), frameReader(t, frm))
	assertLogAndIgnore(t, err)

	resp := <-caller
	var violation runners.ProtocolViolationByServer
	if !errors.As(resp.Err, &violation) {
		t.Fatalf("response err = %v, want ProtocolViolationByServer", resp.Err)
	}
	if _, ok := table.Find(id); ok {
		t.Fatal("request still registered after mismatch")
	}
}

func TestAnswerCount(t *testing.T) {
	if got, err := answerCount(bodies.PingAnswer{}); err != nil || got != 1 {
		t.Fatalf("PingAnswer count = (%d, %v), want (1, nil)", got, err)
	}

	batch := bodies.BatchAnswer{
		{Number: 1, Body: bodies.WriteAnswer{}},
		{Number: 2, Body: bodies.DeleteAnswer{}},
	}
	if got, err := answerCount(batch); err != nil || got != 2 {
		t.Fatalf("BatchAnswer count = (%d, %v), want (2, nil)", got, err)
	}
}

func newActivatingRunner(t *testing.T) (*Runner, *table_v1.RequestsV1, chan []byte) {
	t.Helper()

	table := table_v1.NewRequestsV1()
	requests := make(chan []byte, 8)
	runner, err := NewRunner(table, requests, testRuntimeConfig())
	if err != nil {
		t.Fatalf("NewRunner = %v", err)
	}
	runner.activating.Store(true)
	return runner, table, requests
}

func TestConsumeHandshakeSuccess(t *testing.T) {
	runner, table, _ := newActivatingRunner(t)
	fb := mustBuilder(t)

	frm, err := fb.NewHandshakeAnswer(
		[]fields.Compression{fields.Zstd},
		[]fields.Version{fields.CurrentVersion},
	)
	if err != nil {
		t.Fatalf("NewHandshakeAnswer = %v", err)
	}

	if err := runner.Consume(context.Background(), frameReader(t, frm)); err != nil {
		t.Fatalf("Consume = %v", err)
	}

	if !table.IsGranted() || !table.IsActive() {
		t.Fatalf("flags: granted=%v active=%v", table.IsGranted(), table.IsActive())
	}
	if runner.activating.Load() {
		t.Fatal("activating still set after handshake")
	}
	comps := runner.info.SupportedCompressions()
	if len(comps) != 1 || comps[0] != fields.Zstd {
		t.Fatalf("compressions = %v, want [Zstd]", comps)
	}
	vers := runner.info.SupportedVersions()
	if len(vers) != 1 || vers[0] != fields.CurrentVersion {
		t.Fatalf("versions = %v, want [%v]", vers, fields.CurrentVersion)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady = %v", err)
	}
}

func TestConsumeHandshakeErrorAnswer(t *testing.T) {
	runner, table, _ := newActivatingRunner(t)
	fb := mustBuilder(t)

	frm, err := fb.NewErrAnswer(0, protocolerrs.NewErrorUnauthorized([]byte("x")))
	if err != nil {
		t.Fatalf("NewErrAnswer = %v", err)
	}

	err = runner.Consume(context.Background(), frameReader(t, frm))
	closeErr := assertCloseConnection(t, err)
	var serverErr runners.ServerError
	if !errors.As(closeErr, &serverErr) {
		t.Fatalf("cause = %v, want ServerError", closeErr.Unwrap())
	}
	if table.IsGranted() || table.IsActive() {
		t.Fatal("flags set after handshake error")
	}
	if runner.activating.Load() {
		t.Fatal("activating still set after handshake error")
	}
}

func TestConsumeHandshakeNotActivating(t *testing.T) {
	table := table_v1.NewRequestsV1()
	requests := make(chan []byte, 8)
	runner, err := NewRunner(table, requests, testRuntimeConfig())
	if err != nil {
		t.Fatalf("NewRunner = %v", err)
	}

	frm, err := mustBuilder(t).NewHandshakeAnswer(nil, []fields.Version{1})
	if err != nil {
		t.Fatalf("NewHandshakeAnswer = %v", err)
	}

	err = runner.Consume(context.Background(), frameReader(t, frm))
	assertLogAndIgnore(t, err)
	if table.IsGranted() {
		t.Fatal("granted after ignored handshake")
	}
}

func TestConsumeHandshakeNotAnswer(t *testing.T) {
	runner, _, _ := newActivatingRunner(t)

	frm, err := mustBuilder(t).NewHandshake("user", [32]byte{}, nil)
	if err != nil {
		t.Fatalf("NewHandshake = %v", err)
	}

	err = runner.Consume(context.Background(), frameReader(t, frm))
	closeErr := assertCloseConnection(t, err)
	var violation runners.ProtocolViolationByServer
	if !errors.As(closeErr, &violation) {
		t.Fatalf("cause = %v, want ProtocolViolationByServer", closeErr.Unwrap())
	}
	if runner.activating.Load() {
		t.Fatal("activating still set after violation")
	}
}
