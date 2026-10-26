// Package logging builds the service logger on top of the standard log/slog package.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Supported values for Config.Format.
const (
	FormatJSON = "json"
	FormatText = "text"
)

// Config controls how the logger is built.
type Config struct {
	Level          string // debug, info, warn or error. Empty means info.
	Format         string // json or text. Empty means json.
	ServiceName    string
	ServiceVersion string
}

// ParseLevel converts a level name into a slog.Level. The match is case-insensitive. Empty means info.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q: use debug, info, warn or error", s)
	}
}

// New returns a logger that writes to w. Every record carries service.name and service.version.
// Records logged with a context that holds a valid span also carry trace_id and span_id.
func New(w io.Writer, cfg Config) (*slog.Logger, error) {
	level, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "", FormatJSON:
		handler = slog.NewJSONHandler(w, opts)
	case FormatText:
		handler = slog.NewTextHandler(w, opts)
	default:
		return nil, fmt.Errorf("unsupported log format %q: use json or text", cfg.Format)
	}

	return slog.New(newTraceHandler(handler)).With(
		slog.String("service.name", cfg.ServiceName),
		slog.String("service.version", cfg.ServiceVersion),
	), nil
}
