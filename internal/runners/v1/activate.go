package runner_v1

import (
	"context"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
)

// Activate sends a Handshake (request id 0, not registered in the table) and
// returns. The answer is processed by [Runner.Consume] via consumeHandshake,
// which grants the table and closes [Runner.ready] for [Runner.WaitReady].
//
// If activation is already in progress, the error is [runners.InternalError]
// wrapping [runners.ErrActivating]. If the version is already active, Activate
// is a no-op.
func (r *Runner) Activate(ctx context.Context, login string, hash [32]byte) error {
	if r.activating.Load() {
		return runners.NewInternalError(runners.ErrActivating)
	}
	if r.table.IsActive() {
		return nil
	}

	handshake, err := r.builder.NewHandshake(login, hash, toCompressionsList(r.clientCompressions))
	if err != nil {
		return runners.NewInternalError(err)
	}

	buf := buffer.Slice{}
	buf.Preallocate(handshake.Size())
	if err := handshake.Encode(&buf, nil); err != nil {
		return runners.NewInternalError(err)
	}

	r.activating.Store(true)
	if err := r.send(ctx, buf.Bytes()); err != nil {
		r.activating.Store(false)
		return runners.NewInternalError(err)
	}
	return nil
}
