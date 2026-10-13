package transport

import (
	"context"

	"github.com/dejitarudemon/ignicula-go/values"
)

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
	return nil
}
