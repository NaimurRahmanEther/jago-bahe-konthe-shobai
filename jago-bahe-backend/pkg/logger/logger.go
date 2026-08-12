// Package logger provides the shared structured logger (slog) for the service.
package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON structured logger writing to stdout. Request-ID enrichment
// is added by pkg/httpx middleware (B8 hardens this further).
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler)
}
