package transport

import (
	"bufio"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/logger"
	builderV0 "github.com/dejitarudemon/ignicula-wire/v0/builder"
	decoderV0 "github.com/dejitarudemon/ignicula-wire/v0/decoder"
	fieldsV0 "github.com/dejitarudemon/ignicula-wire/v0/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	builderV1 "github.com/dejitarudemon/ignicula-wire/v1/builder"
	decoderV1 "github.com/dejitarudemon/ignicula-wire/v1/decoder"
	fieldsV1 "github.com/dejitarudemon/ignicula-wire/v1/fields"
	frameV1 "github.com/dejitarudemon/ignicula-wire/v1/frame"
)

func testConnConfig() configs.ConnectionConfig {
	return configs.ConnectionConfig{
		ConnectTimeout: time.Second,
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		MustSupport:    []fieldsV0.Version{1},
		RuntimeV1: configs.RunnerV1Config{
			MaxResponseSize: 1 << 20,
			MaxRequestSize:  1 << 20,
			MaxBatchSize:    64,
			MaxAttempts:     1,
			ReadTimeout:     time.Second,
			WriteTimeout:    time.Second,
			DeleteTimeout:   time.Second,
			PingTimeout:     time.Second,
		},
		Auth: configs.AuthConfig{
			Login: "user",
			Hash:  [32]byte{1, 2, 3},
		},
	}
}

type recLogger struct {
	mu   sync.Mutex
	info []recCall
	warn []recCall
	err  []recCall
}

type recCall struct {
	msg  string
	args []any
}

func (l *recLogger) Debug(string, ...any) {}

func (l *recLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.info = append(l.info, recCall{msg: msg, args: append([]any(nil), args...)})
}

func (l *recLogger) Warn(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warn = append(l.warn, recCall{msg: msg, args: append([]any(nil), args...)})
}

func (l *recLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.err = append(l.err, recCall{msg: msg, args: append([]any(nil), args...)})
}

var _ logger.Logger = (*recLogger)(nil)

func (l *recLogger) infoCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.info)
}

func (l *recLogger) warnCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.warn)
}

func listenTCP(t *testing.T) (addr string, accept <-chan net.Conn, cleanup func()) {
	t.Helper()

	ln, err := net.Listen(NET, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen = %v", err)
	}

	ch := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		ch <- c
	}()

	return ln.Addr().String(), ch, func() { _ = ln.Close() }
}

func waitConn(t *testing.T, accept <-chan net.Conn) net.Conn {
	t.Helper()

	select {
	case c := <-accept:
		return c
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for accept")
		return nil
	}
}

func encodeHello(t *testing.T, versions []fieldsV0.Version) []byte {
	t.Helper()

	frm, err := builderV0.NewFrameBuilder(limitV0).NewHello(versions)
	if err != nil {
		t.Fatalf("NewHello = %v", err)
	}
	buf := buffer.Slice{}
	buf.Preallocate(frm.Size())
	if err := frm.Encode(&buf); err != nil {
		t.Fatalf("Hello Encode = %v", err)
	}
	return buf.Raw()
}

func writeHello(t *testing.T, w io.Writer, versions []fieldsV0.Version) {
	t.Helper()
	if _, err := w.Write(encodeHello(t, versions)); err != nil {
		t.Fatalf("write Hello = %v", err)
	}
}

func encodeV1(t *testing.T, frm frameV1.Frame) []byte {
	t.Helper()

	buf := buffer.Slice{}
	buf.Preallocate(frm.Size())
	if err := frm.Encode(&buf, nil); err != nil {
		t.Fatalf("v1 Encode = %v", err)
	}
	return buf.Bytes()
}

func writeV1(t *testing.T, w io.Writer, frm frameV1.Frame) {
	t.Helper()
	if _, err := w.Write(encodeV1(t, frm)); err != nil {
		t.Fatalf("write v1 = %v", err)
	}
}

func readClientHello(t *testing.T, br *bufio.Reader) {
	t.Helper()

	dec := decoderV0.NewDecoder()
	if _, err := dec.DecodePreamble(br); err != nil {
		t.Fatalf("DecodePreamble Hello = %v", err)
	}
	if _, err := dec.DecodeFrame(br); err != nil {
		t.Fatalf("DecodeFrame Hello = %v", err)
	}
}

func serveHello(t *testing.T, conn net.Conn, versions []fieldsV0.Version) *bufio.Reader {
	t.Helper()

	br := bufio.NewReader(conn)
	readClientHello(t, br)
	writeHello(t, conn, versions)
	return br
}

func readV1Frame(t *testing.T, br *bufio.Reader) frameV1.Frame {
	t.Helper()

	dec := decoderV1.NewDecoder(fieldsV1.BodyLimit(1<<20), nil)
	frm, err := dec.DecodeFrame(br)
	if err != nil {
		t.Fatalf("DecodeFrame v1 = %v", err)
	}
	return frm
}

func grantHandshake(t *testing.T, conn net.Conn, br *bufio.Reader) {
	t.Helper()

	frm := readV1Frame(t, br)
	if frm.Body.Command() != fieldsV1.Handshake {
		t.Fatalf("command = %v, want Handshake", frm.Body.Command())
	}

	answer, err := builderV1.NewFrameBuilder(1 << 20).NewHandshakeAnswer(
		nil,
		[]fieldsV1.Version{fieldsV1.CurrentVersion},
	)
	if err != nil {
		t.Fatalf("NewHandshakeAnswer = %v", err)
	}
	writeV1(t, conn, answer)
}

type peerHello struct {
	conn net.Conn
	br   *bufio.Reader
	err  error
}

func dialOK(t *testing.T, log logger.Logger) (*Connection, net.Conn, *bufio.Reader, func()) {
	t.Helper()

	addr, accept, stopListen := listenTCP(t)
	ready := make(chan peerHello, 1)

	go func() {
		select {
		case c := <-accept:
			br := bufio.NewReader(c)
			dec := decoderV0.NewDecoder()
			if _, err := dec.DecodePreamble(br); err != nil {
				ready <- peerHello{conn: c, err: err}
				return
			}
			if _, err := dec.DecodeFrame(br); err != nil {
				ready <- peerHello{conn: c, err: err}
				return
			}
			frm, err := builderV0.NewFrameBuilder(limitV0).NewHello([]fieldsV0.Version{1})
			if err != nil {
				ready <- peerHello{conn: c, err: err}
				return
			}
			buf := buffer.Slice{}
			buf.Preallocate(frm.Size())
			if err := frm.Encode(&buf); err != nil {
				ready <- peerHello{conn: c, err: err}
				return
			}
			if _, err := c.Write(buf.Raw()); err != nil {
				ready <- peerHello{conn: c, err: err}
				return
			}
			ready <- peerHello{conn: c, br: br}
		case <-time.After(2 * time.Second):
			ready <- peerHello{err: errAcceptTimeout{}}
		}
	}()

	conn, err := NewConnection(addr, testConnConfig(), log)
	if err != nil {
		stopListen()
		t.Fatalf("NewConnection = %v", err)
	}

	peer := <-ready
	if peer.err != nil {
		_ = conn.Close()
		stopListen()
		if peer.conn != nil {
			_ = peer.conn.Close()
		}
		t.Fatalf("peer hello = %v", peer.err)
	}

	cleanup := func() {
		_ = conn.Close()
		_ = peer.conn.Close()
		stopListen()
	}
	return conn, peer.conn, peer.br, cleanup
}

type errAcceptTimeout struct{}

func (errAcceptTimeout) Error() string { return "timed out waiting for accept" }
