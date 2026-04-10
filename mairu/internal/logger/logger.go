package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Config holds logger configuration
type Config struct {
	Level      string
	Structured bool
}

// MultiplexHandler allows writing to multiple handlers
type MultiplexHandler struct {
	handlers []slog.Handler
}

func (m *MultiplexHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiplexHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (m *MultiplexHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiplexHandler{handlers: newHandlers}
}

func (m *MultiplexHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiplexHandler{handlers: newHandlers}
}

var GlobalHandlers []slog.Handler

// Init initializes the global slog logger.
func Init(cfg Config) {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var mainHandler slog.Handler
	if cfg.Structured {
		mainHandler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		mainHandler = slog.NewTextHandler(os.Stderr, opts)
	}

	handlers := []slog.Handler{mainHandler}
	handlers = append(handlers, GlobalHandlers...)

	multi := &MultiplexHandler{handlers: handlers}
	logger := slog.New(multi)
	slog.SetDefault(logger)
}

// Setup is a quick helper to set up standard logging.
func Setup(debug bool) {
	cfg := Config{
		Level:      "info",
		Structured: false,
	}
	if debug {
		cfg.Level = "debug"
	}
	Init(cfg)
}
