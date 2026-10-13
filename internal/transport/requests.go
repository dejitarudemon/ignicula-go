package transport

import (
	"context"
	"errors"

	"github.com/dejitarudemon/ignicula-go/values"
)

// ErrNoActiveVersion is returned when an operation requires an activated
// protocol version but none is active yet.
var ErrNoActiveVersion = errors.New("no active protocol version")

// Read fetches the value stored under key via protocol v1.
// Returns [ErrNoActiveVersion] when the runner is not active yet.
func (c *Connection) Read(ctx context.Context, key []byte) (values.Value, error) {
	if !c.runner_v1.IsActive() {
		return nil, ErrNoActiveVersion
	}
	return c.runner_v1.Read(ctx, key)
}

// Write is not implemented yet.
func (c *Connection) Write(ctx context.Context, key []byte, value values.Value) error {
	return nil
}

// Delete is not implemented yet.
func (c *Connection) Delete(ctx context.Context, key []byte) error {
	return nil
}

// Ping sends a protocol ping via protocol v1.
// Returns [ErrNoActiveVersion] when the runner is not active yet.
func (c *Connection) Ping(ctx context.Context) error {
	if !c.runner_v1.IsActive() {
		return ErrNoActiveVersion
	}

	return c.runner_v1.Ping(ctx)
}
