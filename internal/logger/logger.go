package logger

import (
	"log/slog"
	"os"
)

var defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// L returns the shared logger instance.
func L() *slog.Logger {
	return defaultLogger
}
