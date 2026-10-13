package client

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/internal/transport"
	"github.com/dejitarudemon/ignicula-go/logger"
	"github.com/dejitarudemon/ignicula-go/logger/implemented"
)

type Ignicula struct {
	conn *transport.Connection

	ctx    context.Context
	cancel context.CancelFunc
	err    error

	closeOnce sync.Once
}

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

func (i *Ignicula) Ping(ctx context.Context) error {
	if err := i.err; err != nil {
		return err
	}

	return i.conn.Ping(ctx)
}
