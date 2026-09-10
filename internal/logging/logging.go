// Package logging builds the application's structured logger (log/slog) from
// configuration. Commands construct one logger at start-up and thread it down
// into the router and handlers.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// New builds a *slog.Logger that writes to w. level is one of "debug", "info",
// "warn", or "error" (anything else becomes "info"); format is "json" or
// "text" (anything else becomes "json").
func New(w io.Writer, level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		handler = slog.NewJSONHandler(w, opts)
	}
	return slog.New(handler)
}

func parseLevel(name string) slog.Level {
	switch strings.ToLower(name) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
