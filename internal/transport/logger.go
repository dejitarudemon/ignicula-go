package transport

import (
	"time"
)

// info records msg at info level when a logger is set.
// A nil logger is ignored. args are alternating keys and values.
// Every call adds a "time" field with the current local time.
func (c *Connection) info(msg string, args ...any) {
	if c.logger != nil {
		c.logger.Info(msg, withTime(args)...)
	}
}

// warn records msg at warn level when a logger is set.
// A nil logger is ignored. args are alternating keys and values.
// Every call adds a "time" field with the current local time.
func (c *Connection) warn(msg string, args ...any) {
	if c.logger != nil {
		c.logger.Warn(msg, withTime(args)...)
	}
}

// error records msg at error level when a logger is set.
// A nil logger is ignored. args are alternating keys and values.
// Every call adds a "time" field with the current local time.
func (c *Connection) error(msg string, args ...any) {
	if c.logger != nil {
		c.logger.Error(msg, withTime(args)...)
	}
}

func withTime(args []any) []any {
	return append([]any{"time", time.Now()}, args...)
}
