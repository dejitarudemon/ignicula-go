package transport

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestIsClosedConn(t *testing.T) {
	if !isClosedConn(net.ErrClosed) {
		t.Fatal("net.ErrClosed")
	}
	if !isClosedConn(io.EOF) {
		t.Fatal("io.EOF")
	}
	if !isClosedConn(errors.Join(io.EOF, errors.New("wrap"))) {
		t.Fatal("wrapped EOF")
	}
	if isClosedConn(errors.New("other")) {
		t.Fatal("other error")
	}
}


func TestDeadline(t *testing.T) {
	before := time.Now()
	got := deadline(100 * time.Millisecond)
	after := time.Now()

	if got.Before(before.Add(100 * time.Millisecond)) {
		t.Fatalf("deadline too early: %v", got)
	}
	if got.After(after.Add(100 * time.Millisecond)) {
		t.Fatalf("deadline too late: %v", got)
	}
}

func TestStopTimer_NotFired(t *testing.T) {
	timer := time.NewTimer(time.Hour)
	stopTimer(timer)
	timer.Reset(time.Millisecond)
	select {
	case <-timer.C:
	case <-time.After(time.Second):
		t.Fatal("timer did not fire after Reset")
	}
}

func TestStopTimer_AlreadyFired(t *testing.T) {
	timer := time.NewTimer(time.Millisecond)
	<-timer.C
	stopTimer(timer)
	timer.Reset(time.Millisecond)
	select {
	case <-timer.C:
	case <-time.After(time.Second):
		t.Fatal("timer did not fire after drained stopTimer + Reset")
	}
}
