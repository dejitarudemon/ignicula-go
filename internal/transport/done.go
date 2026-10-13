package transport

import "errors"

// ErrDisconnected is returned when the peer closed the connection or the
// transport aborted due to a read/write failure.
var ErrDisconnected = errors.New("disconnected from server")

// ErrClosed is returned when the connection was closed locally via
// [Connection.Close] and [Connection.Err] is checked after Done is closed
// with no prior disconnect/failure error.
var ErrClosed = errors.New("connection closed")

// Done is closed when the connection is no longer usable (peer disconnect,
// transport failure, or local [Connection.Close]).
func (c *Connection) Done() <-chan struct{} {
	return c.done
}

// Err returns the reason the connection became unusable.
// It is nil until [Connection.Done] is closed. After a clean [Connection.Close]
// with no prior failure it returns [ErrClosed]. After a peer disconnect or
// transport failure it returns an error wrapping [ErrDisconnected].
func (c *Connection) Err() error {
	select {
	case <-c.done:
	default:
		return nil
	}
	if v := c.fatal.Load(); v != nil {
		return v.(error)
	}
	return ErrClosed
}

func (c *Connection) markDone(err error) {
	c.doneOnce.Do(func() {
		if err != nil {
			c.fatal.Store(err)
		}
		close(c.done)
	})
}

// abort records reason and shuts down workers without waiting.
func (c *Connection) abort(reason error) {
	if reason == nil {
		reason = ErrDisconnected
	}
	c.markDone(reason)
	c.closeConn()
}
