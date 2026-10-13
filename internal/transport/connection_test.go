package transport

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	builderV0 "github.com/dejitarudemon/ignicula-wire/v0/builder"
	decoderV0 "github.com/dejitarudemon/ignicula-wire/v0/decoder"
	fieldsV0 "github.com/dejitarudemon/ignicula-wire/v0/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	builderV1 "github.com/dejitarudemon/ignicula-wire/v1/builder"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
)

func TestNewConnection_UnsupportedVersion(t *testing.T) {
	cfg := testConnConfig()
	cfg.MustSupport = []fieldsV0.Version{99}

	_, err := NewConnection("127.0.0.1:1", cfg, nil)
	if err == nil || !strings.Contains(err.Error(), "doesn't support") {
		t.Fatalf("NewConnection = %v, want unsupported version", err)
	}
}

func TestNewConnection_DialFail(t *testing.T) {
	ln, err := net.Listen(NET, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen = %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	_, err = NewConnection(addr, testConnConfig(), nil)
	if err == nil {
		t.Fatal("NewConnection = nil, want dial error")
	}
}

func TestNewConnection_OK(t *testing.T) {
	log := &recLogger{}
	conn, _, _, cleanup := dialOK(t, log)
	defer cleanup()

	if conn.runner_v1 == nil {
		t.Fatal("runner_v1 is nil")
	}
	if log.infoCount() == 0 {
		t.Fatal("expected connect ok log")
	}
}

func TestNewConnection_ServerMissingVersion(t *testing.T) {
	addr, accept, stopListen := listenTCP(t)
	defer stopListen()

	errCh := make(chan error, 1)
	go func() {
		c := <-accept
		defer c.Close()
		br := bufio.NewReader(c)
		dec := decoderV0.NewDecoder()
		if _, err := dec.DecodePreamble(br); err != nil {
			errCh <- err
			return
		}
		if _, err := dec.DecodeFrame(br); err != nil {
			errCh <- err
			return
		}
		frm, err := builderV0.NewFrameBuilder(limitV0).NewHello([]fieldsV0.Version{2})
		if err != nil {
			errCh <- err
			return
		}
		buf := buffer.Slice{}
		buf.Preallocate(frm.Size())
		if err := frm.Encode(&buf); err != nil {
			errCh <- err
			return
		}
		_, err = c.Write(buf.Raw())
		errCh <- err
	}()

	_, err := NewConnection(addr, testConnConfig(), &recLogger{})
	if err == nil || !strings.Contains(err.Error(), "doesn't support") {
		t.Fatalf("NewConnection = %v, want missing version", err)
	}
	if peerErr := <-errCh; peerErr != nil {
		t.Fatalf("peer = %v", peerErr)
	}
}

func TestNewConnection_UnexpectedHelloVersion(t *testing.T) {
	addr, accept, stopListen := listenTCP(t)
	defer stopListen()

	errCh := make(chan error, 1)
	go func() {
		c := <-accept
		defer c.Close()
		br := bufio.NewReader(c)
		dec := decoderV0.NewDecoder()
		if _, err := dec.DecodePreamble(br); err != nil {
			errCh <- err
			return
		}
		if _, err := dec.DecodeFrame(br); err != nil {
			errCh <- err
			return
		}
		frm, err := builderV1.NewFrameBuilder(1 << 20).NewErrAnswer(
			0,
			protocolerrs.NewErrorUnauthorized([]byte("x")),
		)
		if err != nil {
			errCh <- err
			return
		}
		buf := buffer.Slice{}
		buf.Preallocate(frm.Size())
		if err := frm.Encode(&buf, nil); err != nil {
			errCh <- err
			return
		}
		_, err = c.Write(buf.Bytes())
		errCh <- err
	}()

	_, err := NewConnection(addr, testConnConfig(), nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("NewConnection = %v, want unexpected version", err)
	}
	if peerErr := <-errCh; peerErr != nil {
		t.Fatalf("peer = %v", peerErr)
	}
}

func TestNewConnection_HelloTimeout(t *testing.T) {
	addr, accept, stopListen := listenTCP(t)
	defer stopListen()

	go func() {
		c := <-accept
		defer c.Close()
		time.Sleep(300 * time.Millisecond)
	}()

	cfg := testConnConfig()
	cfg.ConnectTimeout = 50 * time.Millisecond

	_, err := NewConnection(addr, cfg, nil)
	if err == nil {
		t.Fatal("NewConnection = nil, want timeout")
	}
}

func TestNewConnection_EmptyHello(t *testing.T) {
	addr, accept, stopListen := listenTCP(t)
	defer stopListen()

	go func() {
		c := <-accept
		defer c.Close()
		br := bufio.NewReader(c)
		dec := decoderV0.NewDecoder()
		_, _ = dec.DecodePreamble(br)
		_, _ = dec.DecodeFrame(br)
		_ = c.Close()
	}()

	_, err := NewConnection(addr, testConnConfig(), nil)
	if err == nil {
		t.Fatal("NewConnection = nil, want read error")
	}
}

func TestClose_WithoutStart(t *testing.T) {
	conn, _, _, cleanup := dialOK(t, nil)
	defer cleanup()

	done := make(chan error, 1)
	go func() { done <- conn.Close() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close blocked without Start")
	}
}

func TestClose_Idempotent(t *testing.T) {
	conn, _, _, cleanup := dialOK(t, nil)
	defer cleanup()

	if err := conn.Close(); err != nil {
		t.Fatalf("Close #1 = %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close #2 = %v", err)
	}
}
