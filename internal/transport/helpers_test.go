package transport

import (
	"testing"
	"time"
)

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
