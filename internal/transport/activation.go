package transport

import (
	"context"
)

// startV1Activation sends Handshake and waits until the runner is granted and
// active. The handshake answer is delivered by the reader via Consume.
func (c *Connection) startV1Activation(ctx context.Context) error {
	if c.runner_v1.IsActive() {
		return nil
	}

	if err := c.runner_v1.Activate(ctx, c.auth.Login, c.auth.Hash); err != nil {
		return err
	}

	return c.WaitReady(ctx)
}

// WaitReady blocks until v1 Handshake succeeds and the runner is active, or
// until ctx / connection shutdown. It is safe to call concurrently with
// [Connection.Start]'s own wait (a closed ready channel unblocks all waiters).
func (c *Connection) WaitReady(ctx context.Context) error {
	if err := c.checkLive(); err != nil {
		return err
	}
	if c.runner_v1.IsActive() {
		return nil
	}
	return c.runner_v1.WaitReady(ctx)
}
