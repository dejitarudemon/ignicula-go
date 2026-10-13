package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"time"
)

func deadline(timeout time.Duration) time.Time {
	return time.Now().Add(timeout)
}

// remainingTimeout returns how long is left on ctx's deadline, or fallback
// when the context has no deadline.
func remainingTimeout(ctx context.Context, fallback time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return time.Until(deadline)
	}
	return fallback
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
