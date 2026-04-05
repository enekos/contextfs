package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds logger configuration
type Config struct {
	Level      string
	Structured bool
}

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

	var handler slog.Handler
	if cfg.Structured {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	logger := slog.New(handler)
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
