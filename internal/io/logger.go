package io

import (
	"io"
	"log/slog"
)

const (
	LevelError LogLevel = "error"
	LevelDebug LogLevel = "debug"
)

type LogLevel string

func (l LogLevel) Level() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	default:
		return slog.LevelError
	}
}

type LogFormat string

// NewLogger returns an slog.Logger for structured logs. Only 'debug' and 'error' log levels are allowed. Super granular
// logging is not needed.
func NewLogger(writer io.Writer, format LogFormat, level LogLevel) *slog.Logger {
	var handler slog.Handler

	switch format {
	case "json":
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	case "text":
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.DiscardHandler
	}

	return slog.New(handler)
}
