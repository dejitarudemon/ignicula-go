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

	return c.runner_v1.WaitReady(ctx)
}
