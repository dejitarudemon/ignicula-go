package configs

import (
	"time"

	"github.com/dejitarudemon/ignicula-wire/v1/compressor"
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
