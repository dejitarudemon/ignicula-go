package transport

import (
	"fmt"
	"net"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/dejitarudemon/ignicula-go/configs"
	runner_v1 "github.com/dejitarudemon/ignicula-go/internal/runners/v1"
	table_v1 "github.com/dejitarudemon/ignicula-go/internal/table/v1"
	"github.com/dejitarudemon/ignicula-go/logger"
	decoderV0 "github.com/dejitarudemon/ignicula-wire/v0/decoder"
	"github.com/dejitarudemon/ignicula-wire/v0/fields"
	fieldsV0 "github.com/dejitarudemon/ignicula-wire/v0/fields"
)

const NET = "tcp"

var (
	supportedVersions = []fieldsV0.Version{1}
)

type Connection struct {
	inner net.Conn

	logger logger.Logger

	versions []fields.Version
	timeouts Timeouts
	auth     configs.AuthConfig

	close     chan struct{}
	closeOnce sync.Once
	started   atomic.Bool

	readerDone chan struct{}
	writerDone chan struct{}

	requests chan []byte

	decoder decoderV0.Decoder

	runner_v1 *runner_v1.Runner
}

func NewConnection(addr string, cfg configs.ConnectionConfig, logger logger.Logger) (*Connection, error) {
	for _, version := range cfg.MustSupport {
		if !slices.Contains(supportedVersions, version) {
			return nil, fmt.Errorf("transport doesn't support %v version", version)
		}
	}

	requests := make(chan []byte)

	runnerV1, err := runner_v1.NewRunner(
		table_v1.NewRequestsV1(),
		requests,
		cfg.RuntimeV1,
	)

	if err != nil {
		return nil, err
	}

	c, err := net.Dial(NET, addr)
	if err != nil {
		return nil, err
	}

	conn := Connection{
		inner:      c,
		logger:     logger,
		versions:   cfg.MustSupport,
		timeouts:   Timeouts{Read: cfg.ReadTimeout, Write: cfg.WriteTimeout},
		close:      make(chan struct{}),
		readerDone: make(chan struct{}),
		writerDone: make(chan struct{}),
		requests:   requests,
		decoder:    decoderV0.NewDecoder(),
		runner_v1:  runnerV1,
		auth:       cfg.Auth,
	}

	if err := conn.hello(cfg.ConnectTimeout); err != nil {
		conn.error("connect to Ignicula server fail", "addr", addr, "error", err)

		_ = c.Close()
		return nil, err
	}

	conn.info("connect to Ignicula server ok", "addr", addr)

	return &conn, nil
}
