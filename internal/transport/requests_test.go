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

func TestWrite_NoActiveVersion(t *testing.T) {
	conn, _, _, cleanup := dialOK(t, nil)
	defer cleanup()

	err := conn.Write(context.Background(), []byte("k"), values.StringValue("v"))
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Fatalf("Write = %v, want ErrNoActiveVersion", err)
	}
}

func TestWrite_Validation(t *testing.T) {
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

	if err := conn.Write(context.Background(), nil, values.StringValue("v")); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("empty key = %v", err)
	}
	if err := conn.Write(context.Background(), []byte("k"), nil); !errors.Is(err, ErrNilValue) {
		t.Fatalf("nil value = %v", err)
	}
}

func TestWrite_OK(t *testing.T) {
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
		done <- conn.Write(context.Background(), []byte("key"), values.StringValue("v"))
	}()

	frm := readV1Frame(t, br)
	if frm.Body.Command() != fieldsV1.Write {
		t.Fatalf("command = %v, want Write", frm.Body.Command())
	}

	answer, err := builderV1.NewFrameBuilder(1 << 20).NewWriteAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewWriteAnswer = %v", err)
	}
	writeV1(t, peer, answer)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Write = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Write did not return")
	}
}

func TestDelete_NoActiveVersion(t *testing.T) {
	conn, _, _, cleanup := dialOK(t, nil)
	defer cleanup()

	err := conn.Delete(context.Background(), []byte("k"))
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Fatalf("Delete = %v, want ErrNoActiveVersion", err)
	}
}

func TestDelete_Validation(t *testing.T) {
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

	if err := conn.Delete(context.Background(), nil); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("empty key = %v", err)
	}
}

func TestDelete_OK(t *testing.T) {
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
		done <- conn.Delete(context.Background(), []byte("key"))
	}()

	frm := readV1Frame(t, br)
	if frm.Body.Command() != fieldsV1.Delete {
		t.Fatalf("command = %v, want Delete", frm.Body.Command())
	}

	answer, err := builderV1.NewFrameBuilder(1 << 20).NewDeleteAnswer(frm.RequestID)
	if err != nil {
		t.Fatalf("NewDeleteAnswer = %v", err)
	}
	writeV1(t, peer, answer)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Delete = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Delete did not return")
	}
}
