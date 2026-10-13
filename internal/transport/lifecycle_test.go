package transport

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	"github.com/dejitarudemon/ignicula-go/logger"
	fieldsV0 "github.com/dejitarudemon/ignicula-wire/v0/fields"
	builderV1 "github.com/dejitarudemon/ignicula-wire/v1/builder"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	fieldsV1 "github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func startGranted(t *testing.T, log *recLogger) (*Connection, func()) {
	t.Helper()

	var l logger.Logger
	if log != nil {
		l = log
	}
	conn, peer, br, cleanup := dialOK(t, l)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		grantHandshake(t, peer, br)
	}()

	ctx, cancel := context.WithCancel(context.Background())

	if err := conn.Start(ctx); err != nil {
		cancel()
		cleanup()
		t.Fatalf("Start = %v", err)
	}
	wg.Wait()

	if !conn.runner_v1.IsActive() {
		cancel()
		cleanup()
		t.Fatal("runner not active after Start")
	}

	return conn, func() {
		cancel()
		cleanup()
	}
}

func TestStart_OK(t *testing.T) {
	log := &recLogger{}
	conn, cleanup := startGranted(t, log)
	defer cleanup()

	if err := conn.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}

	select {
	case <-conn.Done():
	default:
		t.Fatal("Done not closed after Close")
	}
	if err := conn.Err(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Err = %v, want ErrClosed", err)
	}

	log.mu.Lock()
	defer log.mu.Unlock()
	for _, call := range log.err {
		if call.msg == "reader stopped" {
			t.Fatalf("Close logged unexpected error: %q %v", call.msg, call.args)
		}
	}
}

func TestPeerDisconnect_AbortsPending(t *testing.T) {
	conn, peer, br, cleanup := dialOK(t, nil)
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		grantHandshake(t, peer, br)
	}()
	if err := conn.Start(context.Background()); err != nil {
		t.Fatalf("Start = %v", err)
	}
	wg.Wait()

	pingDone := make(chan error, 1)
	go func() {
		pingDone <- conn.Ping(context.Background())
	}()

	// Give ping a moment to register, then peer drops the TCP connection.
	time.Sleep(20 * time.Millisecond)
	_ = peer.Close()

	select {
	case <-conn.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done not closed after peer disconnect")
	}
	if err := conn.Err(); !errors.Is(err, ErrDisconnected) {
		t.Fatalf("Err = %v, want ErrDisconnected", err)
	}

	select {
	case err := <-pingDone:
		if err == nil {
			t.Fatal("Ping = nil after disconnect")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Ping still blocked after peer disconnect")
	}

	if err := conn.Ping(context.Background()); !errors.Is(err, ErrDisconnected) {
		t.Fatalf("Ping after Done = %v, want ErrDisconnected", err)
	}
}

func TestStart_AlreadyStarted(t *testing.T) {
	conn, cleanup := startGranted(t, nil)
	defer cleanup()

	if err := conn.Start(context.Background()); !errors.Is(err, errAlreadyStarted) {
		t.Fatalf("Start = %v, want errAlreadyStarted", err)
	}
}

func TestStartV1Activation_AlreadyActive(t *testing.T) {
	conn, cleanup := startGranted(t, nil)
	defer cleanup()

	if err := conn.startV1Activation(context.Background()); err != nil {
		t.Fatalf("startV1Activation = %v", err)
	}
}

func TestStart_HandshakeRejected(t *testing.T) {
	conn, peer, br, cleanup := dialOK(t, &recLogger{})
	defer cleanup()

	go func() {
		frm := readV1Frame(t, br)
		if frm.Body.Command() != fieldsV1.Handshake {
			t.Errorf("command = %v", frm.Body.Command())
			return
		}
		answer, err := builderV1.NewFrameBuilder(1 << 20).NewErrAnswer(
			0,
			protocolerrs.NewErrorUnauthorized([]byte("denied")),
		)
		if err != nil {
			t.Errorf("NewErrAnswer = %v", err)
			return
		}
		writeV1(t, peer, answer)
	}()

	err := conn.Start(context.Background())
	if err == nil {
		t.Fatal("Start = nil, want handshake error")
	}
	var closeErr runners.CloseConnection
	var serverErr runners.ServerError
	if !errors.As(err, &closeErr) && !errors.As(err, &serverErr) {
		// WaitReady may surface InternalError(ErrRunnerClosed) if reader closes first.
		if !errors.Is(err, runners.ErrRunnerClosed) && !errors.As(err, new(runners.InternalError)) {
			t.Fatalf("Start = %v, want server/close/runner-closed", err)
		}
	}
}

func TestStart_CancelWhileWaiting(t *testing.T) {
	conn, peer, br, cleanup := dialOK(t, nil)
	defer cleanup()
	defer peer.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = readV1Frame(t, br) // consume handshake, never answer
		cancel()
	}()

	err := conn.Start(ctx)
	if err == nil {
		t.Fatal("Start = nil, want cancel error")
	}
}

func TestClose_WaitsForWorkers(t *testing.T) {
	conn, cleanup := startGranted(t, nil)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		_ = conn.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not finish waiting for workers")
	}
}

func TestReader_IgnoresUnsolicitedNonPing(t *testing.T) {
	log := &recLogger{}
	conn, peer, br, cleanup := dialOK(t, log)
	defer cleanup()
	defer peer.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		grantHandshake(t, peer, br)
	}()

	if err := conn.Start(context.Background()); err != nil {
		t.Fatalf("Start = %v", err)
	}
	wg.Wait()

	frm, err := builderV1.NewFrameBuilder(1 << 20).NewWrite(7, fieldsV1.Key("k"), values.String("v"))
	if err != nil {
		t.Fatalf("NewWrite = %v", err)
	}
	writeV1(t, peer, frm)

	deadline := time.Now().Add(time.Second)
	for log.warnCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if log.warnCount() == 0 {
		t.Fatal("expected ignored-frame warn")
	}
}

func TestReader_RepliesServerPing(t *testing.T) {
	conn, peer, br, cleanup := dialOK(t, nil)
	defer cleanup()
	defer peer.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		grantHandshake(t, peer, br)
	}()

	if err := conn.Start(context.Background()); err != nil {
		t.Fatalf("Start = %v", err)
	}
	wg.Wait()

	ping, err := builderV1.NewFrameBuilder(1 << 20).NewPing(42)
	if err != nil {
		t.Fatalf("NewPing = %v", err)
	}
	writeV1(t, peer, ping)

	answer := readV1Frame(t, br)
	if answer.RequestID != 42 {
		t.Fatalf("RequestID = %v, want 42", answer.RequestID)
	}
	if answer.Body.Command() != fieldsV1.Answer {
		t.Fatalf("command = %v, want Answer", answer.Body.Command())
	}
}

func TestReader_UnsupportedVersionCloses(t *testing.T) {
	log := &recLogger{}
	conn, peer, br, cleanup := dialOK(t, log)
	defer cleanup()
	defer peer.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		grantHandshake(t, peer, br)
	}()

	if err := conn.Start(context.Background()); err != nil {
		t.Fatalf("Start = %v", err)
	}
	wg.Wait()

	writeHello(t, peer, []fieldsV0.Version{1})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-conn.close:
			return
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatal("expected connection close after unsupported version")
}

func TestWriter_SkipsEmptyRequest(t *testing.T) {
	conn, cleanup := startGranted(t, nil)
	defer cleanup()

	select {
	case conn.requests <- nil:
	case <-time.After(time.Second):
		t.Fatal("blocked sending empty request")
	}
	select {
	case conn.requests <- []byte{}:
	case <-time.After(time.Second):
		t.Fatal("blocked sending zero-length request")
	}

	time.Sleep(5 * time.Millisecond)
	if err := conn.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}
}

func TestStart_ParentCancelStopsWorkers(t *testing.T) {
	conn, peer, br, cleanup := dialOK(t, nil)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		grantHandshake(t, peer, br)
	}()

	if err := conn.Start(ctx); err != nil {
		t.Fatalf("Start = %v", err)
	}
	wg.Wait()

	cancel()

	done := make(chan struct{})
	go func() {
		_ = conn.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close after parent cancel hung")
	}
}
