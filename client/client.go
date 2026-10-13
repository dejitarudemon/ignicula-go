package client

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/internal/transport"
	"github.com/dejitarudemon/ignicula-go/logger"
	"github.com/dejitarudemon/ignicula-go/logger/implemented"
	"github.com/dejitarudemon/ignicula-go/values"
)

// Re-exported connection lifecycle errors for callers that only import client.
var (
	ErrDisconnected = transport.ErrDisconnected
	ErrClosed       = transport.ErrClosed
)

// Ignicula is a client for an Ignicula server. Dial starts protocol activation
// in the background; Ping, Read, Write, and Delete require a successful
// activation.
type Ignicula struct {
	conn *transport.Connection

	ctx    context.Context
	cancel context.CancelFunc
	err    error

	closeOnce sync.Once
}

// Dial connects to addr, performs the Hello exchange, and starts v1 activation
// asynchronously. A nil log uses a stderr text logger at info level.
func Dial(addr string, cfg configs.ConnectionConfig, log logger.Logger) (*Ignicula, error) {
	if log == nil {
		log = implemented.NewText(slog.LevelInfo)
	}

	conn, err := transport.NewConnection(addr, cfg, log)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	i := &Ignicula{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}

	go func() {
		i.err = conn.Start(ctx)
	}()

	return i, nil
}

// Done is closed when the connection is no longer usable (peer disconnect,
// transport failure, or [Ignicula.Close]).
func (i *Ignicula) Done() <-chan struct{} {
	return i.conn.Done()
}

// Err returns the reason [Ignicula.Done] was closed.
// Before Done is closed it returns nil. See [transport.Connection.Err].
func (i *Ignicula) Err() error {
	if err := i.err; err != nil {
		return err
	}
	return i.conn.Err()
}

// Close cancels the connection context and closes the underlying transport.
// If Start failed, that error is preferred over the close error.
// It is safe to call more than once.
func (i *Ignicula) Close() error {
	var e error
	i.closeOnce.Do(func() {
		i.cancel()
		closeErr := i.conn.Close()
		if err := i.err; err != nil {
			e = err
			return
		}
		e = closeErr
	})

	return e
}

func (i *Ignicula) check() error {
	if err := i.err; err != nil {
		return err
	}
	select {
	case <-i.conn.Done():
		return i.conn.Err()
	default:
		return nil
	}
}

// WaitReady blocks until background activation finishes and the connection can
// serve Ping/Read/Write/Delete, or until ctx is done / the connection fails.
func (i *Ignicula) WaitReady(ctx context.Context) error {
	if err := i.check(); err != nil {
		return err
	}

	err := i.conn.WaitReady(ctx)
	if startErr := i.err; startErr != nil {
		return startErr
	}
	return err
}

// Ping sends a protocol ping. If background Start has already failed, that
// error is returned instead.
func (i *Ignicula) Ping(ctx context.Context) error {
	if err := i.check(); err != nil {
		return err
	}

	return i.conn.Ping(ctx)
}

// Read fetches the value stored under key. If background Start has already
// failed, that error is returned instead.
func (i *Ignicula) Read(ctx context.Context, key []byte) (values.Value, error) {
	if err := i.check(); err != nil {
		return nil, err
	}

	return i.conn.Read(ctx, key)
}

// Write stores data under key using typ. When typ is omitted, [values.Bytes]
// is used. If background Start has already failed, that error is returned
// instead.
func (i *Ignicula) Write(ctx context.Context, key, data []byte, typ ...values.Type) error {
	want := values.Bytes
	if len(typ) > 0 {
		want = typ[0]
	}

	value, err := values.New(want, data)
	if err != nil {
		return err
	}
	return i.WriteValue(ctx, key, value)
}

// WriteValue stores value under key. If background Start has already failed,
// that error is returned instead.
func (i *Ignicula) WriteValue(ctx context.Context, key []byte, value values.Value) error {
	if err := i.check(); err != nil {
		return err
	}

	return i.conn.Write(ctx, key, value)
}

// Delete removes the value stored under key. If background Start has already
// failed, that error is returned instead.
func (i *Ignicula) Delete(ctx context.Context, key []byte) error {
	if err := i.check(); err != nil {
		return err
	}

	return i.conn.Delete(ctx, key)
}
