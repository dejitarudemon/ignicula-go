package transport

import (
	"context"
	"errors"
	"fmt"
)

var errAlreadyStarted = errors.New("connection already started")

// Start launches the reader and writer, then activates protocol v1.
// Workers keep running after Start returns until [Connection.Close], peer
// disconnect, or a transport failure.
func (c *Connection) Start(parent context.Context) error {
	if !c.started.CompareAndSwap(false, true) {
		return errAlreadyStarted
	}

	go func() {
		err := c.reader(parent)
		if c.shuttingDown() || errors.Is(err, context.Canceled) {
			return
		}
		if err == nil {
			return
		}
		if isClosedConn(err) {
			c.info("peer disconnected", "error", err)
			c.abort(fmt.Errorf("%w: %v", ErrDisconnected, err))
			return
		}
		c.error("reader stopped", "error", err)
		c.abort(fmt.Errorf("%w: %v", ErrDisconnected, err))
	}()
	go c.writer(parent)

	if err := c.startV1Activation(parent); err != nil {
		c.error("v1 activation failed", "error", err)
		c.markDone(err)
		_ = c.Close()
		return err
	}

	c.info("v1 activation ok")
	return nil
}
