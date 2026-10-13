package configs

import (
	"testing"
	"time"

	"github.com/dejitarudemon/ignicula-wire/v0/fields"
)

func TestDefaultRunnerV1Config(t *testing.T) {
	cfg := DefaultRunnerV1Config()

	if cfg.MaxResponseSize != 10<<10 || cfg.MaxRequestSize != 10<<10 {
		t.Fatalf("sizes = %d/%d, want 10KiB", cfg.MaxResponseSize, cfg.MaxRequestSize)
	}
	if cfg.MaxBatchSize != 64 {
		t.Fatalf("MaxBatchSize = %d, want 64", cfg.MaxBatchSize)
	}
	if cfg.MaxAttempts != 3 {
		t.Fatalf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.Backoff != 100*time.Millisecond {
		t.Fatalf("Backoff = %v", cfg.Backoff)
	}
	if cfg.ReadTimeout != 30*time.Second || cfg.WriteTimeout != 30*time.Second || cfg.DeleteTimeout != 30*time.Second {
		t.Fatalf("op timeouts = %v/%v/%v", cfg.ReadTimeout, cfg.WriteTimeout, cfg.DeleteTimeout)
	}
	if cfg.PingTimeout != 5*time.Second {
		t.Fatalf("PingTimeout = %v", cfg.PingTimeout)
	}
	if cfg.Compressions != nil {
		t.Fatalf("Compressions = %v, want nil", cfg.Compressions)
	}
}

func TestDefaultConnectionConfig(t *testing.T) {
	cfg := DefaultConnectionConfig()

	if cfg.ConnectTimeout != 30*time.Second || cfg.ReadTimeout != 100*time.Second || cfg.WriteTimeout != 100*time.Second {
		t.Fatalf("timeouts = %v/%v/%v", cfg.ConnectTimeout, cfg.ReadTimeout, cfg.WriteTimeout)
	}
	if len(cfg.MustSupport) != 1 || cfg.MustSupport[0] != fields.Version(1) {
		t.Fatalf("MustSupport = %v, want [1]", cfg.MustSupport)
	}
	wantRuntime := DefaultRunnerV1Config()
	if cfg.RuntimeV1.MaxResponseSize != wantRuntime.MaxResponseSize ||
		cfg.RuntimeV1.MaxBatchSize != wantRuntime.MaxBatchSize ||
		cfg.RuntimeV1.PingTimeout != wantRuntime.PingTimeout {
		t.Fatalf("RuntimeV1 = %+v, want defaults", cfg.RuntimeV1)
	}
	if cfg.Auth != (AuthConfig{}) {
		t.Fatalf("Auth = %+v, want empty", cfg.Auth)
	}
}

func TestDefaultAuthConfig(t *testing.T) {
	if got := DefaultAuthConfig(); got != (AuthConfig{}) {
		t.Fatalf("DefaultAuthConfig = %+v, want empty", got)
	}
}
