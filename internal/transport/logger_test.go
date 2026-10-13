package transport

import (
	"testing"
	"time"
)

func TestLoggerNilIsNoop(t *testing.T) {
	c := &Connection{}
	c.info("info")
	c.warn("warn")
	c.error("error")
}

func TestLoggerRecordsWithTime(t *testing.T) {
	log := &recLogger{}
	c := &Connection{logger: log}

	c.info("hello", "k", 1)
	c.warn("careful", "k", 2)
	c.error("boom", "k", 3)

	log.mu.Lock()
	defer log.mu.Unlock()

	checks := []struct {
		name string
		call recCall
	}{
		{"info", log.info[0]},
		{"warn", log.warn[0]},
		{"error", log.err[0]},
	}
	for _, tt := range checks {
		if tt.call.msg == "" {
			t.Fatalf("%s: empty msg", tt.name)
		}
		if len(tt.call.args) < 4 {
			t.Fatalf("%s: args = %v, want time + payload", tt.name, tt.call.args)
		}
		if tt.call.args[0] != "time" {
			t.Fatalf("%s: first key = %v, want time", tt.name, tt.call.args[0])
		}
		if _, ok := tt.call.args[1].(time.Time); !ok {
			t.Fatalf("%s: time value type = %T", tt.name, tt.call.args[1])
		}
	}
}

func TestWithTimePrepends(t *testing.T) {
	got := withTime([]any{"a", 1})
	if len(got) != 4 || got[0] != "time" || got[2] != "a" || got[3] != 1 {
		t.Fatalf("withTime = %v", got)
	}
	if _, ok := got[1].(time.Time); !ok {
		t.Fatalf("time type = %T", got[1])
	}
}
