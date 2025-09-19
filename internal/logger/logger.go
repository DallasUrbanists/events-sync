package logger

import (
	"log/slog"
	"os"
	"strings"
)

var LEVELS = map[string]slog.Level{
	"DEBUG": slog.LevelDebug,
	"INFO": slog.LevelInfo,
	"WARN": slog.LevelWarn,
	"ERROR": slog.LevelError,
}

func NewLogger() *slog.Logger {
	level, ok := LEVELS[strings.ToUpper(os.Getenv("LOG_LEVEL"))]
	if !ok {
		level = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
