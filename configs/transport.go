package configs

import (
	"time"

	"github.com/dejitarudemon/ignicula-wire/v0/fields"
)

type ConnectionConfig struct {
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration

	MustSupport []fields.Version

	RuntimeV1 RunnerV1Config
	Auth      AuthConfig
}
