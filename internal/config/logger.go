package config

import (
	"log/slog"
	"os"
	"strings"
)

func (c *LoggerConfig) NewLogger() *slog.Logger {
	level := parseLogLevel(c.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
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
