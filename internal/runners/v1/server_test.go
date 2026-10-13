package runner_v1

import (
	"slices"
	"sync"
	"testing"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

func TestNewServerInfo(t *testing.T) {
	s := NewServerInfo()

	if s.BodyLimit() != 0 || s.BatchLimit() != 0 {
		t.Fatalf("limits = (%v, %v), want zeros", s.BodyLimit(), s.BatchLimit())
	}
	if s.SupportedCompressions() != nil {
		t.Fatalf("compressions = %v, want nil", s.SupportedCompressions())
	}
	if s.SupportedVersions() != nil {
		t.Fatalf("versions = %v, want nil", s.SupportedVersions())
	}
}

func TestServerInfoSetAndGet(t *testing.T) {
	s := NewServerInfo()

	s.SetBodyLimit(1024)
	s.SetBatchLimit(64)
	s.SetSupportedCompressions([]fields.Compression{fields.Zstd, fields.S2})
	s.SetSupportedVersions([]fields.Version{1, 2})

	if s.BodyLimit() != 1024 {
		t.Fatalf("BodyLimit() = %v, want 1024", s.BodyLimit())
	}
	if s.BatchLimit() != 64 {
		t.Fatalf("BatchLimit() = %v, want 64", s.BatchLimit())
	}
	if got := s.SupportedCompressions(); !slices.Equal(got, []fields.Compression{fields.Zstd, fields.S2}) {
		t.Fatalf("SupportedCompressions() = %v", got)
	}
	if got := s.SupportedVersions(); !slices.Equal(got, []fields.Version{1, 2}) {
		t.Fatalf("SupportedVersions() = %v", got)
	}
}

func TestServerInfoSnapshots(t *testing.T) {
	s := NewServerInfo()
	inCompressions := []fields.Compression{fields.Zstd}
	inVersions := []fields.Version{1}

	s.SetSupportedCompressions(inCompressions)
	s.SetSupportedVersions(inVersions)

	inCompressions[0] = fields.S2
	inVersions[0] = 9

	if got := s.SupportedCompressions(); !slices.Equal(got, []fields.Compression{fields.Zstd}) {
		t.Fatalf("Set kept caller slice: %v", got)
	}
	if got := s.SupportedVersions(); !slices.Equal(got, []fields.Version{1}) {
		t.Fatalf("Set kept caller slice: %v", got)
	}

	outCompressions := s.SupportedCompressions()
	outVersions := s.SupportedVersions()
	outCompressions[0] = fields.None
	outVersions[0] = 7

	if got := s.SupportedCompressions(); !slices.Equal(got, []fields.Compression{fields.Zstd}) {
		t.Fatalf("getter snapshot mutated store: %v", got)
	}
	if got := s.SupportedVersions(); !slices.Equal(got, []fields.Version{1}) {
		t.Fatalf("getter snapshot mutated store: %v", got)
	}
}

func TestServerInfoSetReplaces(t *testing.T) {
	s := NewServerInfo()
	s.SetSupportedCompressions([]fields.Compression{fields.Zstd, fields.S2})
	s.SetSupportedVersions([]fields.Version{1, 2})

	s.SetSupportedCompressions([]fields.Compression{fields.None})
	s.SetSupportedVersions(nil)
	s.SetBodyLimit(0)
	s.SetBatchLimit(0)

	if got := s.SupportedCompressions(); !slices.Equal(got, []fields.Compression{fields.None}) {
		t.Fatalf("compressions = %v, want [None]", got)
	}
	if s.SupportedVersions() != nil {
		t.Fatalf("versions = %v, want nil", s.SupportedVersions())
	}
	if s.BodyLimit() != 0 || s.BatchLimit() != 0 {
		t.Fatalf("limits = (%v, %v), want zeros", s.BodyLimit(), s.BatchLimit())
	}
}

func TestServerInfoConcurrent(t *testing.T) {
	s := NewServerInfo()
	var wg sync.WaitGroup

	wg.Go(func() {
		for i := range 100 {
			s.SetBodyLimit(fields.BodyLimit(i))
			s.SetBatchLimit(fields.BatchLimit(i))
			s.SetSupportedCompressions([]fields.Compression{fields.Zstd})
			s.SetSupportedVersions([]fields.Version{1})
		}
	})
	wg.Go(func() {
		for range 100 {
			_ = s.BodyLimit()
			_ = s.BatchLimit()
			_ = s.SupportedCompressions()
			_ = s.SupportedVersions()
		}
	})

	wg.Wait()
}

func TestNilServerInfoPanics(t *testing.T) {
	var s *ServerInfo

	tests := []struct {
		name string
		call func()
	}{
		{"body limit", func() { _ = s.BodyLimit() }},
		{"batch limit", func() { _ = s.BatchLimit() }},
		{"compressions", func() { _ = s.SupportedCompressions() }},
		{"versions", func() { _ = s.SupportedVersions() }},
		{"set body", func() { s.SetBodyLimit(1) }},
		{"set batch", func() { s.SetBatchLimit(1) }},
		{"set compressions", func() { s.SetSupportedCompressions(nil) }},
		{"set versions", func() { s.SetSupportedVersions(nil) }},
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
