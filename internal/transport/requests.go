package transport

import (
	"context"
	"errors"

	"github.com/dejitarudemon/ignicula-go/values"
)

// ErrNoActiveVersion is returned when an operation requires an activated
// protocol version but none is active yet.
var ErrNoActiveVersion = errors.New("no active protocol version")

// ErrEmptyKey is returned when a key argument is empty.
var ErrEmptyKey = errors.New("empty key")

// ErrNilValue is returned when a write value is nil.
var ErrNilValue = errors.New("nil value")

func (c *Connection) checkLive() error {
	select {
	case <-c.done:
		return c.Err()
	default:
		return nil
	}
}

// Read fetches the value stored under key via protocol v1.
// Returns [ErrNoActiveVersion] when the runner is not active yet.
func (c *Connection) Read(ctx context.Context, key []byte) (values.Value, error) {
	if err := c.checkLive(); err != nil {
		return nil, err
	}
	if !c.runner_v1.IsActive() {
		return nil, ErrNoActiveVersion
	}
	if len(key) == 0 {
		return nil, ErrEmptyKey
	}
	return c.runner_v1.Read(ctx, key)
}

// Write stores value under key via protocol v1.
// Returns [ErrNoActiveVersion] when the runner is not active yet.
func (c *Connection) Write(ctx context.Context, key []byte, value values.Value) error {
	if err := c.checkLive(); err != nil {
		return err
	}
	if !c.runner_v1.IsActive() {
		return ErrNoActiveVersion
	}
	if len(key) == 0 {
		return ErrEmptyKey
	}
	if value == nil {
		return ErrNilValue
	}
	return c.runner_v1.Write(ctx, key, value)
}

// Delete removes the value stored under key via protocol v1.
// Returns [ErrNoActiveVersion] when the runner is not active yet.
func (c *Connection) Delete(ctx context.Context, key []byte) error {
	if err := c.checkLive(); err != nil {
		return err
	}
	if !c.runner_v1.IsActive() {
		return ErrNoActiveVersion
	}
	if len(key) == 0 {
		return ErrEmptyKey
	}
	return c.runner_v1.Delete(ctx, key)
}

// Ping sends a protocol ping via protocol v1.
// Returns [ErrNoActiveVersion] when the runner is not active yet.
func (c *Connection) Ping(ctx context.Context) error {
	if err := c.checkLive(); err != nil {
		return err
	}
	if !c.runner_v1.IsActive() {
		return ErrNoActiveVersion
	}

	return c.runner_v1.Ping(ctx)
}
