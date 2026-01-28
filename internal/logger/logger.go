package logger

import (
	"log/slog"
	"os"
)

// InitLogger initializes and configures the application logger based on environment
// Returns a configured slog.Logger instance
func InitLogger(environment string) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// In development, use more verbose logging and text handler
	if environment == "development" {
		opts.Level = slog.LevelDebug
		opts.AddSource = true // Include source file and line number
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// In production, use JSON handler for structured logging
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	
	// Set as default logger so it can be used throughout the application
	slog.SetDefault(logger)

	return logger
}
