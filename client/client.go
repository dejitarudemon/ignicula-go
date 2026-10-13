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

// Ignicula is a client for an Ignicula server. Dial starts protocol activation
// in the background; Ping and Read require a successful activation.
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

// Ping sends a protocol ping. If background Start has already failed, that
// error is returned instead.
func (i *Ignicula) Ping(ctx context.Context) error {
	if err := i.err; err != nil {
		return err
	}

	return i.conn.Ping(ctx)
}

// Read fetches the value stored under key. If background Start has already
// failed, that error is returned instead.
func (i *Ignicula) Read(ctx context.Context, key []byte) (values.Value, error) {
	if err := i.err; err != nil {
		return nil, err
	}

	return i.conn.Read(ctx, key)
}
