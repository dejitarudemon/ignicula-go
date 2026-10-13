package configs

import (
	"time"

	"github.com/dejitarudemon/ignicula-wire/v1/compressor"
)

const (
	// Default body/request size matches the server runtime body limit (10 KiB).
	defaultMaxResponseSize = 10 << 10
	defaultMaxRequestSize  = 10 << 10
	// Default batch size matches the server runtime batch limit (64).
	defaultMaxBatchSize = 1 << 6
	defaultMaxAttempts  = 3
	defaultBackoff      = 100 * time.Millisecond
	defaultOpTimeout    = 30 * time.Second
	defaultPingTimeout  = 5 * time.Second
)

type RunnerV1Config struct {
	MaxResponseSize int
	MaxRequestSize  int
	MaxBatchSize    int

	MaxAttempts int
	Backoff     time.Duration

	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	DeleteTimeout time.Duration
	PingTimeout   time.Duration

	Compressions []compressor.Compressor
}

// DefaultRunnerV1Config returns runner settings aligned with common server
// limits: 10 KiB frames, 64-ops batches, 3 attempts with 100ms backoff, 30s
// CRUD timeouts, and a 5s ping timeout. No compressors are registered.
func DefaultRunnerV1Config() RunnerV1Config {
	return RunnerV1Config{
		MaxResponseSize: defaultMaxResponseSize,
		MaxRequestSize:  defaultMaxRequestSize,
		MaxBatchSize:    defaultMaxBatchSize,
		MaxAttempts:     defaultMaxAttempts,
		Backoff:         defaultBackoff,
		ReadTimeout:     defaultOpTimeout,
		WriteTimeout:    defaultOpTimeout,
		DeleteTimeout:   defaultOpTimeout,
		PingTimeout:     defaultPingTimeout,
	}
}
