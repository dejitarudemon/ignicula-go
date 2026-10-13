package implemented

import (
	"log/slog"
	"os"

	"github.com/dejitarudemon/ignicula-go/logger"
)

// Logger forwards [logger.Logger] calls to a [slog.Logger].
type Logger struct {
	s *slog.Logger
}

// New wraps s as a [logger.Logger]. A nil s uses [slog.Default].
func New(s *slog.Logger) *Logger {
	if s == nil {
		s = slog.Default()
	}
	return &Logger{s: s}
}

// NewText returns a stderr text logger at level. A nil level uses [slog.LevelInfo].
func NewText(level slog.Leveler) *Logger {
	if level == nil {
		level = slog.LevelInfo
	}
	return New(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}

var _ logger.Logger = (*Logger)(nil)

func (l *Logger) Debug(msg string, args ...any) { l.s.Debug(msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.s.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.s.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.s.Error(msg, args...) }
