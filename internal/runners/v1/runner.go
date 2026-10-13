package runner_v1

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/internal/runners"
	table_v1 "github.com/dejitarudemon/ignicula-go/internal/table/v1"
	"github.com/dejitarudemon/ignicula-wire/v1/builder"
	"github.com/dejitarudemon/ignicula-wire/v1/compressor"
	"github.com/dejitarudemon/ignicula-wire/v1/decoder"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

type Runner struct {
	decoder decoder.Decoder
	builder builder.FrameBuilder

	info *ServerInfo

	table *table_v1.RequestsV1

	requests chan<- []byte

	timeouts Timeouts

	ready      chan struct{}
	activating atomic.Bool

	close     chan struct{}
	closeOnce sync.Once

	clientCompressions map[fields.Compression]compressor.Compressor
}

func NewRunner(table *table_v1.RequestsV1, requests chan<- []byte, cfg configs.RunnerV1Config) (*Runner, error) {
	if requests == nil {
		return nil, runners.ErrNilRequestsChan
	}

	if table == nil {
		return nil, runners.ErrNilRequestsV1
	}

	cfg.MaxAttempts = max(1, cfg.MaxAttempts)

	compressors := toCompressionsMap(cfg.Compressions)

	runner := Runner{
		decoder:            decoder.NewDecoder(fields.BodyLimit(cfg.MaxResponseSize), nil),
		builder:            builder.NewFrameBuilder(cfg.MaxRequestSize),
		info:               NewServerInfo(),
		table:              table,
		requests:           requests,
		ready:              make(chan struct{}),
		close:              make(chan struct{}),
		clientCompressions: compressors,
		timeouts: Timeouts{
			Read:        cfg.ReadTimeout,
			Write:       cfg.WriteTimeout,
			Delete:      cfg.DeleteTimeout,
			Ping:        cfg.PingTimeout,
			Backoff:     cfg.Backoff,
			MaxAttempts: cfg.MaxAttempts,
		},
	}
	runner.info.SetBatchLimit(fields.BatchLimit(cfg.MaxBatchSize))
	return &runner, nil
}

// WaitReady blocks until Handshake succeeds ([Runner.ready] is closed) or ctx /
// the runner is cancelled. A closed runner returns [runners.ErrRunnerClosed].
func (r *Runner) WaitReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.close:
		return runners.NewInternalError(runners.ErrRunnerClosed)
	case _, ok := <-r.ready:
		if !ok {
			return nil
		}
	}

	return runners.NewInternalError(runners.ErrWaitReady)
}

// Close signals shutdown: unblocks send/await/WaitReady, and terminates all
// registered requests. It is safe to call more than once.
func (r *Runner) Close() {
	r.closeOnce.Do(func() {
		close(r.close)
		r.table.TerminateAll()
	})
}

func (r *Runner) IsActive() bool {
	return r.table.IsActive()
}
