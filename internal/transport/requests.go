package transport

import (
	"context"
	"errors"

	"github.com/dejitarudemon/ignicula-go/values"
)

// ErrNoActiveVersion is returned when an operation requires an activated
// protocol version but none is active yet.
var ErrNoActiveVersion = errors.New("no active protocol version")

func (c *Connection) Read(ctx context.Context, key []byte) (values.Value, error) {
	return nil, nil
}

func (c *Connection) Write(ctx context.Context, key []byte, value values.Value) error {
	return nil
}

func (c *Connection) Delete(ctx context.Context, key []byte) error {
	return nil
}

func (c *Connection) Ping(ctx context.Context) error {
	if !c.runner_v1.IsActive() {
		return ErrNoActiveVersion
	}

	return c.runner_v1.Ping(ctx)
}
