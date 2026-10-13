package transport

// Close shuts down workers, the v1 runner, and the underlying connection, then
// waits until reader and writer have exited (if Start was called). It is safe
// to call more than once.
func (c *Connection) Close() error {
	return c.shutdown(true)
}

// closeConn signals shutdown from a worker goroutine without waiting on that
// worker (avoids deadlock on readerDone/writerDone).
func (c *Connection) closeConn() {
	if err := c.shutdown(false); err != nil {
		c.error("failed to close connection", "error", err)
	}
}

func (c *Connection) shutdown(wait bool) error {
	var err error
	c.closeOnce.Do(func() {
		close(c.close)
		c.runner_v1.Close()
		err = c.inner.Close()
	})

	if wait && c.started.Load() {
		<-c.readerDone
		<-c.writerDone
	}

	return err
}
