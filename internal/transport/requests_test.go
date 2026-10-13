package transport

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	builderV1 "github.com/dejitarudemon/ignicula-wire/v1/builder"
	fieldsV1 "github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func TestRequestStubs(t *testing.T) {
	c := &Connection{}
	ctx := context.Background()

	if v, err := c.Read(ctx, []byte("k")); v != nil || err != nil {
		t.Fatalf("Read = (%v, %v), want (nil, nil)", v, err)
	}
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
