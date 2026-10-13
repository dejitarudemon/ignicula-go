package configs

import (
	"time"

	"github.com/dejitarudemon/ignicula-wire/v0/fields"
)

const (
	defaultConnectTimeout = 30 * time.Second
	defaultReadTimeout    = 100 * time.Second
	defaultWriteTimeout   = 100 * time.Second
)

type ConnectionConfig struct {
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration

	MustSupport []fields.Version

	RuntimeV1 RunnerV1Config
	Auth      AuthConfig
}

// DefaultConnectionConfig returns connection settings with a 30s dial timeout,
// 100s read/write deadlines (server idle-ping window), MustSupport version 1,
// and [DefaultRunnerV1Config]. Auth is left empty and must be set by the caller
// before connecting.
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		ConnectTimeout: defaultConnectTimeout,
		ReadTimeout:    defaultReadTimeout,
		WriteTimeout:   defaultWriteTimeout,
		MustSupport:    []fields.Version{1},
		RuntimeV1:      DefaultRunnerV1Config(),
	}
}
