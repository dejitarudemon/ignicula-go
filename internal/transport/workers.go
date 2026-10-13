package transport

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	"github.com/dejitarudemon/ignicula-wire/v0/fields"
)

const (
	// netBufferSize is the max number of encoded answer frames coalesced into
	// one writev before a forced flush.
	netBufferSize = 16
	// flushAfter is the production latency ceiling for a partial batch.
	// Without it, a non-full batch would wait for more answers indefinitely.
	flushAfter = time.Microsecond
)

func (c *Connection) reader(ctx context.Context) error {
	defer close(c.readerDone)

	reader := bufio.NewReader(c.inner)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.close:
			return nil
		default:
		}

		if err := c.inner.SetDeadline(deadline(c.timeouts.Read)); err != nil {
			return err
		}

		version, err := c.decoder.DecodePreamble(reader)
		if err != nil {
			return err
		}

		switch version {
		case fields.Version(1):
			if err := c.runner_v1.Consume(ctx, reader); err != nil {
				var ignore runners.LogAndIgnore
				if errors.As(err, &ignore) {
					c.warn("ignored frame", "error", err)
					continue
				}
				return err
			}
		default:
			return fmt.Errorf("unsupported protocol version %v", version)
		}
	}
}

func (c *Connection) writer(ctx context.Context) {
	defer close(c.writerDone)

	batch := make(net.Buffers, 0, netBufferSize)
	timer := time.NewTimer(flushAfter)
	stopTimer(timer)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.close:
			return

		case request := <-c.requests:
			if len(request) == 0 {
				continue
			}
			batch = append(batch, request)
			if len(batch) < cap(batch) {
				if len(batch) == 1 {
					timer.Reset(flushAfter)
				}
				continue
			}

		case <-timer.C:
			if len(batch) == 0 {
				continue
			}
		}

		if err := c.inner.SetDeadline(deadline(c.timeouts.Write)); err != nil {
			c.error("failed to set write deadline", "error", err)
			c.closeConn()
			return
		}

		if _, err := batch.WriteTo(c.inner); err != nil {
			c.error("failed to request", "error", err)
			c.closeConn()
			return
		}
		batch = batch[:0]
		stopTimer(timer)
	}
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
