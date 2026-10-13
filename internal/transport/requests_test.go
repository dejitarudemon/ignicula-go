package transport

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-go/values"
	builderV1 "github.com/dejitarudemon/ignicula-wire/v1/builder"
	fieldsV1 "github.com/dejitarudemon/ignicula-wire/v1/fields"
	protovalues "github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func TestRequestStubs(t *testing.T) {
	c := &Connection{}
	ctx := context.Background()

	if err := c.Write(ctx, []byte("k"), nil); err != nil {
		t.Fatalf("Write = %v", err)
	}
	if err := c.Delete(ctx, []byte("k")); err != nil {
		t.Fatalf("Delete = %v", err)
	}
}

func TestPing_NoActiveVersion(t *testing.T) {
	conn, _, _, cleanup := dialOK(t, nil)
	defer cleanup()

	err := conn.Ping(context.Background())
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Fatalf("Ping = %v, want ErrNoActiveVersion", err)
	}
}

func TestPing_OK(t *testing.T) {
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

	done := make(chan error, 1)
	go func() {
		done <- conn.Ping(context.Background())
	}()

	frm := readV1Frame(t, br)
	if frm.Body.Command() != fieldsV1.Ping {
		t.Fatalf("command = %v, want Ping", frm.Body.Command())
	}

	answer, err := builderV1.NewFrameBuilder(1 << 20).NewPingAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewPingAnswer = %v", err)
	}
	writeV1(t, peer, answer)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Ping = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Ping did not return")
	}
}

func TestRead_NoActiveVersion(t *testing.T) {
	conn, _, _, cleanup := dialOK(t, nil)
	defer cleanup()

	_, err := conn.Read(context.Background(), []byte("k"))
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Fatalf("Read = %v, want ErrNoActiveVersion", err)
	}
}

func TestRead_OK(t *testing.T) {
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

	type result struct {
		val values.Value
		err error
	}
	done := make(chan result, 1)
	go func() {
		val, err := conn.Read(context.Background(), []byte("key"))
		done <- result{val: val, err: err}
	}()

	frm := readV1Frame(t, br)
	if frm.Body.Command() != fieldsV1.Read {
		t.Fatalf("command = %v, want Read", frm.Body.Command())
	}

	answer, err := builderV1.NewFrameBuilder(1 << 20).NewReadAnswer(frm.RequestID, protovalues.String("v"))
	if err != nil {
		t.Fatalf("NewReadAnswer = %v", err)
	}
	writeV1(t, peer, answer)

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Read = %v", got.err)
		}
		sv, ok := got.val.(values.StringValue)
		if !ok || string(sv) != "v" {
			t.Fatalf("Read value = %#v, want StringValue(v)", got.val)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not return")
	}
}
