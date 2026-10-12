package table

import (
	"errors"
	"testing"

	"github.com/dejitarudemon/ignicula-go/transport/table/versions"
	"github.com/dejitarudemon/ignicula-wire/v0/fields"
)

func TestNewRequestsTable(t *testing.T) {
	table := NewRequestsTable()

	if table.V1() == nil {
		t.Fatal("V1() = nil")
	}
	if table.V1().IsGranted() || table.V1().IsActive() {
		t.Fatal("new table v1 is already granted or active")
	}
	if !table.Support(Version1) {
		t.Fatal("Support(Version1) = false")
	}
}

func TestSupportOnlyVersion1(t *testing.T) {
	table := NewRequestsTable()

	tests := []struct {
		version fields.Version
		want    bool
	}{
		{0, false},
		{Version1, true},
		{1, true},
		{2, false},
		{255, false},
	}

	for _, tt := range tests {
		if got := table.Support(tt.version); got != tt.want {
			t.Errorf("Support(%d) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestCloseTerminatesV1AndClearsFlags(t *testing.T) {
	table := NewRequestsTable()
	v1 := table.V1()
	v1.Grant(true)
	v1.Activate(true)

	id, err := v1.RegisterOutcomingRequest(1)
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest = %v", err)
	}

	table.Close()

	if _, ok := v1.Find(id); ok {
		t.Fatal("Find after Close = true, want terminated")
	}
	if v1.IsGranted() || v1.IsActive() {
		t.Fatalf("Close left flags set: granted=%v active=%v", v1.IsGranted(), v1.IsActive())
	}

	_, err = v1.RegisterOutcomingRequest(1)
	if !errors.Is(err, versions.ErrVersionNeitherGrantedNorActivated) {
		t.Fatalf("RegisterOutcomingRequest after Close = %v, want not ready", err)
	}

	v1.Grant(true)
	v1.Activate(true)
	id, err = v1.RegisterOutcomingRequest(1)
	if err != nil {
		t.Fatalf("RegisterOutcomingRequest after re-grant = %v", err)
	}
	if _, ok := v1.Find(id); !ok {
		t.Fatal("v1 not usable after re-grant")
	}
}

func TestNilRequestsTablePanics(t *testing.T) {
	var table *RequestsTable

	tests := []struct {
		name string
		call func()
	}{
		{"close", func() { table.Close() }},
		{"v1", func() { _ = table.V1() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			tt.call()
		})
	}
}

func TestNilSupportDoesNotPanic(t *testing.T) {
	var table *RequestsTable
	if !table.Support(Version1) {
		t.Fatal("nil Support(Version1) = false")
	}
	if table.Support(2) {
		t.Fatal("nil Support(2) = true")
	}
}
