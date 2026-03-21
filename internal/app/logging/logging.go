package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New builds the process logger used by API and worker entrypoints.
func New(level string) *slog.Logger {
	return NewWithWriter(level, os.Stdout)
}

// NewWithWriter builds the process logger with a caller-provided sink.
func NewWithWriter(level string, writer io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: parseLevel(level),
	}))
}

func parseLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
