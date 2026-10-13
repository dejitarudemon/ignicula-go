package transport

import (
	"context"
	"errors"
)

var errAlreadyStarted = errors.New("connection already started")

// Start launches the reader and writer, then activates protocol v1.
// Workers keep running after Start returns until [Connection.Close] or parent
// cancellation.
func (c *Connection) Start(parent context.Context) error {
	if !c.started.CompareAndSwap(false, true) {
		return errAlreadyStarted
	}

	go func() {
		if err := c.reader(parent); err != nil && !errors.Is(err, context.Canceled) {
			c.error("reader stopped", "error", err)
			c.closeConn()
		}
	}()
	go c.writer(parent)

	if err := c.startV1Activation(parent); err != nil {
		c.error("v1 activation failed", "error", err)
		_ = c.Close()
		return err
	}

	c.info("v1 activation ok")
	return nil
}
