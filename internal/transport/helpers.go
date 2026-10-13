package transport

import (
	"errors"
	"io"
	"net"
	"time"
)

func deadline(timeout time.Duration) time.Time {
	return time.Now().Add(timeout)
}

// shuttingDown reports whether [Connection.Close] / [Connection.closeConn]
// has already started (the close channel is closed).
func (c *Connection) shuttingDown() bool {
	select {
	case <-c.close:
		return true
	default:
		return false
	}
}

// isClosedConn reports whether err is a local closed-connection or clean EOF
// from the peer, including wrapped decode/op errors.
func isClosedConn(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe)
}
